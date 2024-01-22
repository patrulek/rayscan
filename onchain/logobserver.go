package onchain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"github.com/patrulek/rayscan/connection"
	"github.com/patrulek/rayscan/onchain/raydium"
	"github.com/patrulek/rayscan/onchain/serum"
)

type LogSet struct {
	logs map[solana.Signature]struct{} // Tx signature -> empty struct
	mu   sync.RWMutex
}

var logset *LogSet = &LogSet{
	logs: make(map[solana.Signature]struct{}),
}

type LogObserver struct {
	rpcPool *connection.RPCPool

	stopC chan struct{}
	doneC []chan struct{}

	running atomic.Bool

	mu       sync.RWMutex
	connName string
}

func NewLogObserver(rpcPool *connection.RPCPool, connName string) *LogObserver {
	return &LogObserver{
		rpcPool:  rpcPool,
		stopC:    make(chan struct{}),
		doneC:    make([]chan struct{}, 0),
		running:  atomic.Bool{},
		connName: connName,
	}
}

func (o *LogObserver) ConnectionName() string {
	return o.connName
}

func (o *LogObserver) Start(ctx context.Context, txCandidatePublishC chan<- TxCandidate) error {
	if !o.running.CompareAndSwap(false, true) {
		return fmt.Errorf("LogObserver is already running")
	}

	openBookSubID, err := o.subscribeForOpenBookLogs(ctx)
	if err != nil {
		return err
	}

	raydiumSubID, err := o.subscribeForRaydiumLogs(ctx)
	if err != nil {
		openBookSubID.Unsubscribe()
		return err
	}

	go o.consumeOpenBookLogs(openBookSubID, txCandidatePublishC)
	go o.consumeRaydiumLogs(raydiumSubID, txCandidatePublishC)

	return nil
}

func (o *LogObserver) subscribeForOpenBookLogs(ctx context.Context) (*ws.LogSubscription, error) {
	fmt.Printf("[%v] LogObserver: Subscribe for OpenBook program logs on %s...\n", time.Now().Format("2006-01-02 15:04:05.000"), o.connName)
	conn := o.rpcPool.NamedConnection(o.connName)
	wsClient, err := ws.Connect(ctx, conn.ConnectionInfo.WSEndpoint)
	if err != nil {
		return nil, err
	}

	return wsClient.LogsSubscribeMentions(serum.OpenBookDex, rpc.CommitmentProcessed)
}

func (o *LogObserver) consumeOpenBookLogs(openBookSubID *ws.LogSubscription, txCandidatePublishC chan<- TxCandidate) {
	o.mu.Lock()
	doneCIdx := len(o.doneC)
	o.doneC = append(o.doneC, make(chan struct{}))
	o.mu.Unlock()

	defer close(o.doneC[doneCIdx])

	for o.running.Load() {
		log, err := openBookSubID.Recv()
		if err != nil {
			o.reconnectOpenBookSubscription(openBookSubID, err)
			continue
		}

		if log.Value.Logs == nil || log.Value.Err != nil {
			continue // Skip this message.
		}

		logset.mu.RLock()
		_, ok := logset.logs[log.Value.Signature]
		logset.mu.RUnlock()

		if ok {
			continue // Already processed.
		}

		go o.analyzeOpenBookLogs(log, txCandidatePublishC)
	}
}

func (o *LogObserver) reconnectOpenBookSubscription(subID *ws.LogSubscription, reason error) {
	fmt.Printf("[%v] LogObserver: Reconnecting subscription for OpenBook program logs on %s due to: %v...\n", time.Now().Format("2006-01-02 15:04:05.000"), o.connName, reason)
	subID.Unsubscribe()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	openBookSubID, err := o.subscribeForOpenBookLogs(ctx)
	for err != nil {
		fmt.Printf("[%v] LogObserver: Error reconnecting subscription for OpenBook program logs on %s: %v; trying again...\n", time.Now().Format("2006-01-02 15:04:05.000"), o.connName, err)
		time.Sleep(5 * time.Second)
		ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
		openBookSubID, err = o.subscribeForOpenBookLogs(ctx)
		cancel()
	}

	*subID = *openBookSubID
}

func (o *LogObserver) analyzeOpenBookLogs(log *ws.LogResult, txCandidatePublishC chan<- TxCandidate) {
	// Find possible InitMarket instruction logs:
	for i := range log.Value.Logs {
		curLog := log.Value.Logs[i]
		if !strings.Contains(curLog, "Program 11111111111111111111111111111111 success") {
			continue // Search further.
		}

		if i+1 >= len(log.Value.Logs) {
			break // No more logs.
		}

		nextLog := log.Value.Logs[i+1]
		if !strings.Contains(nextLog, "Program srmqPvymJeFKQ4zGQed1GFppgkRHL9kaELCbyksJtPX invoke [1]") {
			continue // Search further.
		}

		// Found it: send signature with no metadata
		txCandidate := TxCandidate{log.Value.Signature, o.connName, nil}
		txCandidatePublishC <- txCandidate
		break
	}

	logset.mu.Lock()
	logset.logs[log.Value.Signature] = struct{}{}
	logset.mu.Unlock()
}

func (o *LogObserver) subscribeForRaydiumLogs(ctx context.Context) (*ws.LogSubscription, error) {
	fmt.Printf("[%v] LogObserver: Subscribe for Raydium Liquidity program logs on %s...\n", time.Now().Format("2006-01-02 15:04:05.000"), o.connName)
	conn := o.rpcPool.NamedConnection(o.connName)
	wsClient, err := ws.Connect(ctx, conn.ConnectionInfo.WSEndpoint)
	if err != nil {
		return nil, err
	}

	return wsClient.LogsSubscribeMentions(raydium.Raydium_Liquidity_Program_V4, rpc.CommitmentProcessed)
}

func (o *LogObserver) consumeRaydiumLogs(raydiumSubID *ws.LogSubscription, txCandidatePublishC chan<- TxCandidate) {
	o.mu.Lock()
	doneCIdx := len(o.doneC)
	o.doneC = append(o.doneC, make(chan struct{}))
	o.mu.Unlock()

	defer close(o.doneC[doneCIdx])

	for o.running.Load() {
		log, err := raydiumSubID.Recv()
		if err != nil {
			o.reconnecRaydiumSubscription(raydiumSubID, err)
			continue
		}

		if log.Value.Logs == nil || log.Value.Err != nil {
			continue // Skip this message.
		}

		logset.mu.RLock()
		_, ok := logset.logs[log.Value.Signature]
		logset.mu.RUnlock()

		if ok {
			continue // Already processed.
		}

		go o.analyzeRaydiumLogs(log, txCandidatePublishC)
	}
}

func (o *LogObserver) reconnecRaydiumSubscription(subID *ws.LogSubscription, reason error) {
	fmt.Printf("[%v] LogObserver: Reconnecting subscription for Raydium program logs on %s due to: %v...\n", time.Now().Format("2006-01-02 15:04:05.000"), o.connName, reason)
	subID.Unsubscribe()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	raydiumSubID, err := o.subscribeForRaydiumLogs(ctx)
	if err != nil {
		fmt.Printf("[%v] LogObserver: Error reconnecting subscription for Raydium program logs on %s: %v; trying again...\n", time.Now().Format("2006-01-02 15:04:05.000"), o.connName, err)
		time.Sleep(5 * time.Second)
		ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
		raydiumSubID, err = o.subscribeForRaydiumLogs(ctx)
		cancel()
	}

	*subID = *raydiumSubID
}

func (o *LogObserver) analyzeRaydiumLogs(log *ws.LogResult, txCandidatePublishC chan<- TxCandidate) {
	// Raw input data; bytes[2:6] -> possible timestamp for ido openning; bytes[7:15] -> possible pc amount, last 8 bytes, possible coin amount
	// Find possible Purchase IDO instruction logs:
	for i := range log.Value.Logs {
		curLog := log.Value.Logs[i]

		// Parse IDO info from log
		_, after, found := strings.Cut(curLog, " InitializeInstruction2 ")
		if !found {
			continue // Search further, not IDO log.
		}

		// Add quotes to keys.
		splitted := strings.Split(after, " ")
		for i, s := range splitted {
			if strings.Contains(s, ":") {
				splitted[i] = "\"" + s[:len(s)-1] + "\":"
			}
		}

		metadata := json.RawMessage(strings.Join(splitted, " "))
		if !json.Valid(metadata) {
			continue // Search further, invalid JSON.
		}

		// Found it: send signature with metadata
		txCandidate := TxCandidate{log.Value.Signature, o.connName, &metadata}
		txCandidatePublishC <- txCandidate
		break
	}

	logset.mu.Lock()
	logset.logs[log.Value.Signature] = struct{}{}
	logset.mu.Unlock()
}

func (o *LogObserver) Stop(ctx context.Context) error {
	if !o.running.CompareAndSwap(true, false) {
		return fmt.Errorf("LogObserver is not running")
	}

	close(o.stopC)
	doneCount := 0

	for doneCount < len(o.doneC) {
		select {
		case <-ctx.Done():
			fmt.Printf("Err: LogObserver: forced shutdown\n")
			return ctx.Err()
		case <-o.doneC[doneCount]:
			doneCount++
		}
	}

	return nil
}
