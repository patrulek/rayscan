package onchain

import (
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/patrulek/rayscan/onchain/raydium"
	"github.com/patrulek/rayscan/onchain/serum"
)

type TokenInfo struct {
	TimeToSerumMarket    time.Duration
	TxCountToSerumMarket uint64
	TotalSupply          uint64
	Decimals             uint8
	TxID                 solana.Signature
	TxTime               time.Time
	Address              solana.PublicKey
}

func (t *TokenInfo) TokenAddress() solana.PublicKey {
	return t.Address
}

func (t *TokenInfo) Ready() bool {
	return t.TimeToSerumMarket != 0
}

type PairInfo struct {
	// MarketInfo is the info about the market.
	MarketInfo serum.MarketInfo

	// TokenInfo is the metadata of token related to serum market info.
	TokenInfo TokenInfo

	// raydium.AmmInfo is the info about the AMM.
	AmmInfo           raydium.AmmInfo
	CalculatedAmmInfo raydium.AmmInfo

	// PairInfo metadata.
	Readiness time.Time // Timestamp of when the first swap is ready to be executed.

	mu sync.RWMutex
}

func (p *PairInfo) GetCurrentAmmLiveInfo() raydium.AmmLiveInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.AmmInfo.CurrentLiveInfo
}

func (p *PairInfo) SetCurrentAmmLiveInfo(ammLiveInfo raydium.AmmLiveInfo) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	p.AmmInfo.CurrentLiveInfo = ammLiveInfo
}

var AnalosPairInfo = &PairInfo{
	MarketInfo: serum.MarketInfo{
		Market:      solana.MustPublicKeyFromBase58("3sJVHtBTjpHmTgArbtTz5cDr6umJZUTjx8yZfyoGAhZm"),
		Bids:        solana.MustPublicKeyFromBase58("6v5VTf216NTojE1fqTjGneDo9Gv4VRWpnFHZ6CwcLE6N"),
		Asks:        solana.MustPublicKeyFromBase58("FVUiBevqdWPYawWF71wy7guzM99NqcXkezSuNW9ZH1Xs"),
		EventQueue:  solana.MustPublicKeyFromBase58("EPwYeGjPb3vJbd3wPN4d5oPEVtwoufKeJ7kktXYMgsrV"),
		BaseVault:   solana.MustPublicKeyFromBase58("Dd5CeV5pfBAkQUimseZ9psrqpe9BwF6LhGA4nGhBbski"),
		QuoteVault:  solana.MustPublicKeyFromBase58("9gWd9qjVsdXemKLXZQ1zNHhPNo3WsYZcxYBTWDZLMs7X"),
		VaultSigner: solana.MustPublicKeyFromBase58("CyZn7qBwL9cMfxh1ghDsqSQwqardaEaq8sienzxqBg5x"),
	},
	AmmInfo: raydium.AmmInfo{
		AmmID:                solana.MustPublicKeyFromBase58("69grLw4PcSypZnn3xpsozCJFT8vs8WA5817VUVnzNGTh"),
		AmmAuthority:         solana.MustPublicKeyFromBase58("5Q544fKrFoe6tsEbD7S8EmxGTJYAKtTVhAW5Q5pge4j1"),
		AmmOpenOrders:        solana.MustPublicKeyFromBase58("2QoiVyXa8Bfgx35yTcJhQJVvZYouKZutj5r5CEDPgQUm"),
		AmmTargetOrders:      solana.MustPublicKeyFromBase58("4m5ecbRZzY7G7TaksDFmv2u8x52FLd9msemHC3xhtFWg"),
		PoolCoinTokenAccount: solana.MustPublicKeyFromBase58("9ibeYfpgDxyoSYNvsc37EwGua6Z1NQpp7LH6e7CvBabM"),
		PoolPcTokenAccount:   solana.MustPublicKeyFromBase58("5JFikPKzw3JeXKaJZaKTEAHfF3pJAoFJbHXUB5p2Ns5S"),
		TokenMintAddress:     solana.MustPublicKeyFromBase58("7iT1GRYYhEop2nV1dyCwK2MGyLmPHq47WhPGSwiqcUg5"),
		CurrencyAddress:      solana.SolMint,
	},
	Readiness: time.Now(),
}

func (p *PairInfo) TokenAddress() solana.PublicKey {
	return p.MarketInfo.TokenAddress()
}

func (p *PairInfo) Ready() bool {
	return p.MarketInfo.Ready() && p.AmmInfo.Ready() && p.TokenInfo.Ready()
}
