package onchain

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/patrulek/rayscan/connection"
	"github.com/patrulek/rayscan/onchain/raydium"
	"github.com/patrulek/rayscan/onchain/serum"
)

var Max_Transaction_Version uint64 = 1
var Rewards bool = false

type TxCandidate struct {
	Signature  solana.Signature
	clientName string
	Metadata   *json.RawMessage
}

type TxAnalyzer struct {
	rpcPool *connection.RPCPool

	txCandidateC chan TxCandidate
	doneC        chan struct{}
	infoPublishC chan<- Info

	gotCandidates map[solana.Signature]struct{}
}

func NewTxAnalyzer(rpcPool *connection.RPCPool) *TxAnalyzer {
	return &TxAnalyzer{
		rpcPool:       rpcPool,
		txCandidateC:  make(chan TxCandidate, 32),
		doneC:         make(chan struct{}),
		gotCandidates: make(map[solana.Signature]struct{}),
	}
}

func (a *TxAnalyzer) Channel() chan<- TxCandidate {
	return a.txCandidateC
}

func (a *TxAnalyzer) Start(infoPublishC chan<- Info) {
	fmt.Printf("[%v] TxAnalyzer: starting...\n", time.Now().Format("2006-01-02 15:04:05.000"))

	go func() {
		defer close(a.doneC)

		for txCandidate := range a.txCandidateC {
			if _, ok := a.gotCandidates[txCandidate.Signature]; ok {
				continue
			}

			a.gotCandidates[txCandidate.Signature] = struct{}{}

			go a.analyze(txCandidate, infoPublishC)
		}
	}()
}

func (a *TxAnalyzer) analyze(txCandidate TxCandidate, infoPublishC chan<- Info) {
	// Send rpc for full tx details
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	rpcTx, tx, err := a.getConfirmedTransaction(ctx, txCandidate)
	if err != nil {
		fmt.Printf("[%v] TxAnalyzer: error getting transaction (tx: %s): %s\n", time.Now().Format("2006-01-02 15:04:05.000"), txCandidate.Signature, err)
		return
	}

	// Known for sure that empty metadata is InitializeMarket instruction.
	if txCandidate.Metadata == nil {
		if err := a.analyzeInitMarket(rpcTx, tx, txCandidate, infoPublishC); err != nil {
			fmt.Printf("[%v] TxAnalyzer: error analyzing init market (tx: %s): %s\n", time.Now().Format("2006-01-02 15:04:05.000"), txCandidate.Signature, err)
		}
		return
	}

	if err := a.analyzeAddLiquidity(rpcTx, tx, txCandidate, infoPublishC); err != nil {
		fmt.Printf("[%v] TxAnalyzer: error analyzing add liquidity (tx: %s): %s\n", time.Now().Format("2006-01-02 15:04:05.000"), txCandidate.Signature, err)
	}
}

func (a *TxAnalyzer) getConfirmedTransaction(ctx context.Context, txCandidate TxCandidate) (*rpc.GetTransactionResult, *solana.Transaction, error) {
	rpcClient := a.rpcPool.NamedConnection(txCandidate.clientName).RPCClient

	for {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
			rctx, rcancel := context.WithTimeout(ctx, 5*time.Second)
			rpcTx, err := rpcClient.GetTransaction(rctx, txCandidate.Signature, &rpc.GetTransactionOpts{
				MaxSupportedTransactionVersion: &Max_Transaction_Version,
				Commitment:                     rpc.CommitmentConfirmed,
			})
			rcancel()

			if err != nil {
				rpcClient = a.rpcPool.Client() // Try with another client.
				continue
			}

			if rpcTx.Meta.Err != nil {
				return nil, nil, fmt.Errorf("Transaction failed: %v", rpcTx.Meta.Err)
			}

			tx, err := rpcTx.Transaction.GetTransaction()
			if err != nil {
				return nil, nil, fmt.Errorf("Couldnt get transaction: %v", err)
			}

			return rpcTx, tx, nil
		}
	}
}

func (a *TxAnalyzer) analyzeInitMarket(rpcTx *rpc.GetTransactionResult, tx *solana.Transaction, txCandidate TxCandidate, infoPublishC chan<- Info) error {
	minfo, err := serum.MarketInfoFromTransaction(rpcTx, tx)
	if err != nil {
		return fmt.Errorf("error getting market info: %w", err)
	}

	infoPublishC <- &minfo

	ainfo, err := raydium.DeriveAmmInfoFromMarket(minfo)
	if err != nil {
		return fmt.Errorf("error deriving amm info from market: %w", err)
	}

	infoPublishC <- ainfo

	tinfo, err := a.TokenInfoFromMarket(minfo)
	if err != nil {
		return fmt.Errorf("error getting token info from market: %w", err)
	}

	infoPublishC <- &tinfo

	return nil
}

func (a *TxAnalyzer) TokenInfoFromMarket(market serum.MarketInfo) (TokenInfo, error) {
	Limit := 100
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	sigs, err := a.rpcPool.Client().GetSignaturesForAddressWithOpts(ctx, market.TokenAddress(),
		&rpc.GetSignaturesForAddressOpts{
			Limit: &Limit,
		},
	)
	if err != nil || len(sigs) == 0 {
		return TokenInfo{}, err
	}

	lastSig := sigs[len(sigs)-1]
	if lastSig.BlockTime == nil {
		return TokenInfo{}, fmt.Errorf("no block time in signature")
	}

	var createTime time.Time
	if lastSig.BlockTime != nil {
		createTime = lastSig.BlockTime.Time()
	}

	rpcTokenSupply, err := a.rpcPool.Client().GetTokenSupply(ctx, market.TokenAddress(), rpc.CommitmentFinalized)
	if err != nil {
		return TokenInfo{}, err
	}

	if rpcTokenSupply.Value == nil {
		return TokenInfo{}, fmt.Errorf("no token supply")
	}

	uval, err := strconv.ParseUint(rpcTokenSupply.Value.Amount, 10, 64)
	if err != nil {
		return TokenInfo{}, err
	}

	tinfo := TokenInfo{
		Address:              market.TokenAddress(),
		TxID:                 lastSig.Signature,
		TxTime:               createTime,
		TimeToSerumMarket:    market.TxTime.Sub(createTime),
		TxCountToSerumMarket: uint64(len(sigs)),
		TotalSupply:          uval,
		Decimals:             rpcTokenSupply.Value.Decimals,
	}

	return tinfo, nil
}

func (a *TxAnalyzer) analyzeAddLiquidity(rpcTx *rpc.GetTransactionResult, tx *solana.Transaction, txCandidate TxCandidate, infoPublishC chan<- Info) error {
	// raydium.AmmInfo instruction
	ainfo, err := raydium.AmmInfoFromTransaction(rpcTx, tx, txCandidate.Metadata)
	if err != nil {
		return fmt.Errorf("error getting amm info: %w", err)
	}

	infoPublishC <- &ainfo

	return nil
}

func (a *TxAnalyzer) Stop(ctx context.Context) error {
	close(a.txCandidateC)

	select {
	case <-a.doneC:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
