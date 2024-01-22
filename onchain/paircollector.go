package onchain

import (
	"context"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/patrulek/rayscan/onchain/raydium"
	"github.com/patrulek/rayscan/onchain/serum"
)

type Info interface {
	// Ready returns true if the info is ready to be used.
	Ready() bool
	TokenAddress() solana.PublicKey
}

type PairCollector struct {
	infoC chan Info
	doneC chan struct{}

	// Key is BaseMint (Token) address as it exists in both MarketInfo and raydium.AmmInfo.
	pairs        map[solana.PublicKey]*PairInfo
	createdPairs map[solana.PublicKey]struct{}

	dropAmmWithoutMarket bool
	dropLowLiquidity     bool
}

func NewPairCollector() *PairCollector {
	return &PairCollector{
		infoC:                make(chan Info, 32),
		doneC:                make(chan struct{}),
		pairs:                make(map[solana.PublicKey]*PairInfo),
		createdPairs:         make(map[solana.PublicKey]struct{}),
		dropAmmWithoutMarket: true,
	}
}

func (c *PairCollector) Channel() chan<- Info {
	return c.infoC
}

func (c *PairCollector) Start(pairPublishC []chan<- *PairInfo) {
	fmt.Printf("[%v] PairCollector: starting...\n", time.Now().Format("2006-01-02 15:04:05.000"))

	go func() {
		defer close(c.doneC)

		for genericInfo := range c.infoC {
			tokenAddress := genericInfo.TokenAddress()
			_, cOk := c.createdPairs[tokenAddress]
			if cOk {
				fmt.Printf("[%v] PairCollector: already created pair for token: %s\n", time.Now().Format("2006-01-02 15:04:05.000"), tokenAddress)
				continue
			}

			pair, err := c.handleInfo(genericInfo, pairPublishC)
			if err != nil {
				fmt.Printf("[%v] PairCollector: error handling info (%T): %s\n", time.Now().Format("2006-01-02 15:04:05.000"), genericInfo, err)
				continue
			}

			if pair.AmmInfo.TxID.IsZero() {
				continue // Wait until amm info arrive
			}

			if !pair.Ready() {
				fmt.Printf("[%v] PairCollector: pair got all info but not ready; drop it (token: %s, ammid: %s)\n", time.Now().Format("2006-01-02 15:04:05.000"), tokenAddress, pair.AmmInfo.AmmID)
				delete(c.pairs, tokenAddress)
				continue
			}

			pair.Readiness = time.Now()

			// Update pair status.
			if tokenAddress != solana.WrappedSol {
				c.createdPairs[tokenAddress] = struct{}{}
				fmt.Printf("[%v] PairCollector: new pair found (token: %s, ammid: %s, opentime: %s)\n", time.Now().Format("2006-01-02 15:04:05.000"), tokenAddress, pair.AmmInfo.AmmID, pair.AmmInfo.InitialLiveInfo.UpdateTime.Format("2006-01-02 15:04:05.000"))
				delete(c.pairs, tokenAddress)
			}
		}
	}()
}

func (c *PairCollector) handleInfo(genericInfo Info, pairPublishC []chan<- *PairInfo) (*PairInfo, error) {
	switch info := genericInfo.(type) {
	case *serum.MarketInfo:
		return c.handleMarketInfo(info)
	case *raydium.AmmInfo:
		return c.handleAmmInfo(info)
	case *TokenInfo:
		return c.handleTokenInfo(info)
	default:
		panic(fmt.Errorf("unknown info type: %T", info))
	}
}

func (c *PairCollector) handleMarketInfo(market *serum.MarketInfo) (*PairInfo, error) {
	tokenAddress := market.TokenAddress()
	_, ok := c.pairs[tokenAddress]
	if ok {
		return nil, fmt.Errorf("pair already exists for token: %s", tokenAddress)
	}

	c.pairs[tokenAddress] = &PairInfo{
		MarketInfo: *market,
	}

	fmt.Printf("[%v] PairCollector: new market discovered for (token: %s, tx time: %v)\n", time.Now().Format("2006-01-02 15:04:05.000"), tokenAddress, market.TxTime.Format("2006-01-02 15:04:05.000"))
	return c.pairs[tokenAddress], nil
}

func (c *PairCollector) handleAmmInfo(amm *raydium.AmmInfo) (*PairInfo, error) {
	tokenAddress := amm.TokenAddress()
	pair, ok := c.pairs[tokenAddress]
	ammSwapped := false

	// Derived amm info should always be right so just assign.
	if amm.Calculated {
		pair.CalculatedAmmInfo = *amm
		return pair, nil
	}

	// In case token address is swapped, swap it back.
	if tokenAddress == solana.WrappedSol {
		tokenAddress = amm.CurrencyAddress // Addresses are swapped but we dont know it yet, because didnt sync with market info. Lets swap it manually.
		pair, ok = c.pairs[tokenAddress]
		ammSwapped = true
	}

	if !ok {
		return nil, fmt.Errorf("amm, no pair for token: %s", tokenAddress)
	}

	amm.UpdateSwap(ammSwapped)
	pair.AmmInfo = *amm

	if !pair.AmmInfo.PoolCoinTokenAccount.Equals(pair.CalculatedAmmInfo.PoolCoinTokenAccount) {
		fmt.Printf("[%v] PairCollector: pool coin token account mismatch (token: %s, calctoken: %s, ammid: %s, poolcoin: %s, calcpoolcoin: %s, poolpc: %s, calcpoolpc: %s, coinamount: %f, pcamount: %f)\n",
			time.Now().Format("2006-01-02 15:04:05.000"), tokenAddress, pair.CalculatedAmmInfo.TokenAddress(), pair.AmmInfo.AmmID, pair.AmmInfo.PoolCoinTokenAccount, pair.CalculatedAmmInfo.PoolCoinTokenAccount,
			pair.AmmInfo.PoolPcTokenAccount, pair.CalculatedAmmInfo.PoolPcTokenAccount, pair.AmmInfo.InitialLiveInfo.PooledToken, pair.AmmInfo.InitialLiveInfo.PooledLamports)
		pair.AmmInfo.PoolCoinTokenAccount = pair.CalculatedAmmInfo.PoolCoinTokenAccount
		pair.AmmInfo.PoolPcTokenAccount = pair.CalculatedAmmInfo.PoolPcTokenAccount
	}

	return pair, nil
}

func (c *PairCollector) handleTokenInfo(token *TokenInfo) (*PairInfo, error) {
	tokenAddress := token.TokenAddress()
	pair, ok := c.pairs[tokenAddress]
	if !ok {
		return nil, fmt.Errorf("token, no pair for token: %s", tokenAddress)
	}

	pair.TokenInfo = *token

	return pair, nil
}

func (c *PairCollector) Stop(ctx context.Context) error {
	close(c.infoC)

	select {
	case <-c.doneC:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
