package raydium

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/patrulek/rayscan/onchain/serum"
)

var (
	Raydium_Liquidity_Program_V4 solana.PublicKey = solana.MustPublicKeyFromBase58("675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8") // This program calls Raydium Purchase IDO to create a new pair.
	Raydium_Authority_Program_V4 solana.PublicKey = solana.MustPublicKeyFromBase58("5Q544fKrFoe6tsEbD7S8EmxGTJYAKtTVhAW5Q5pge4j1") // This is also a wallet that holds tokens and do swaps.
)

// Raydium Purchase Ido: https://solscan.io/tx/5keDz6sQMZWWZurg82htZHd4HmpWjCScYbUvmvcSJtjCTHXt8FRMzAEBNZgbQ3v3pir9ATyPpqPHjqAUKTqodWkr
// https://explorer.solana.com/tx/5keDz6sQMZWWZurg82htZHd4HmpWjCScYbUvmvcSJtjCTHXt8FRMzAEBNZgbQ3v3pir9ATyPpqPHjqAUKTqodWkr

// Transaction version: 0

// programID -> Raydium AMM (Liquidity)
// account1 -> TokenProgram
// account2 -> AssociatedTokenProgram
// account3 -> SystemProgram
// account4 -> Sysvar: Rent
// account5 -> IdoID -> AmmId
// account6 -> Raydium Authority
// account7 -> PoolQuoteTokenAccount -> AmmOpenOrders
// account8 -> UserQuoteTokenAccount (LP token address)
// account9 -> ..BaseMint -> UserIdoInfo -> (unused TokenAddress)
// account10 -> ..QuoteMint -> UserOwner -> (unused CurrencyAddress, zawszse WSOL)
// account11 -> UserStakeInfo -> PoolCoinTokenAccount
// account12 -> UserIdoCheck -> PoolPcTokenAccount
// account13 -> ?? -> AmmTargetOrders
// account14 -> ?? unused ??
// account15 -> ?? unused ??
// account16 -> OpenBook program
// account17 -> ..Market -> .. -> SerumMarket
// account18 -> Singer(Wallet)
// account19 -> ?? unused ??
// account20 -> ?? AmmSolDstAccount ??
// account21 -> LP Token Destination Account
type AmmInfo struct {
	// Purchase IDO Instruction Data
	ProgramID            solana.PublicKey // Raydium Liquidity Pool V4 Program ID
	AmmID                solana.PublicKey // Amm ID (Pair Address)
	AmmAuthority         solana.PublicKey // Amm Authority (Raydium Authority)
	AmmOpenOrders        solana.PublicKey // Amm Open Orders (PoolQuoteTokenAccount)
	LPTokenAddress       solana.PublicKey // LPToken Address (PoolTokenMint)
	TokenMintAddress     solana.PublicKey // Token Address (TokenMint)
	CurrencyAddress      solana.PublicKey // Currency Address (always WSOLMint)
	PoolCoinTokenAccount solana.PublicKey // Amm Token Account (PoolCoinTokenAccount)
	PoolPcTokenAccount   solana.PublicKey // Amm WSOL Token Account (PoolPcTokenAccount)
	AmmTargetOrders      solana.PublicKey // Amm Target Orders
	AmmLiquidityCreator  solana.PublicKey // Amm Liquidity Creator (ata account of LP creator that will receive LP tokens)
	Calculated           bool

	// Purchase IDO Instruction Metadata
	Caller    solana.PublicKey // Caller wallet address
	TxID      solana.Signature // Transaction ID
	Slot      uint64           // Chain Slot
	TxTime    time.Time        // Timestamp of transaction in blockchain
	Timestamp time.Time        // Timestamp of transaction discovery

	// Purchase IDO Instruction Log
	InitialLiveInfo AmmLiveInfo

	// Raydium Pool Live Info
	CurrentLiveInfo AmmLiveInfo // Current live info (taken from RPC)
}

type AmmLiveInfo struct {
	UpdateTime     time.Time // Amm trading open time (taken from instruction log); for initial it will be OpenTime of the market.
	PooledLamports float64   // Current pooled WSOL
	PooledToken    float64   // Current pooled MintToken
	Price          float64   // Current price (Pr := MintToken/WSOL)
	LPTokenBurned  bool      // Whether LP tokens were burned (false = LP tokens were not burned or unknown)
	MintDisabled   bool      // Whether minting is disabled (false = minting is enabled or unknown)
}

func (a *AmmLiveInfo) Ready() bool {
	return a.UpdateTime != time.Time{} && a.PooledLamports != 0 && a.PooledToken != 0 && a.Price != 0
}

func NewAmmInfo() *AmmInfo {
	return &AmmInfo{
		ProgramID:       Raydium_Liquidity_Program_V4,
		AmmAuthority:    Raydium_Authority_Program_V4,
		CurrencyAddress: solana.WrappedSol,
	}
}

func (a *AmmInfo) Ready() bool {
	return a.AmmID != solana.PublicKey{} && a.AmmOpenOrders != solana.PublicKey{} && a.LPTokenAddress != solana.PublicKey{} &&
		a.TokenMintAddress != solana.PublicKey{} && a.PoolCoinTokenAccount != solana.PublicKey{} && a.PoolPcTokenAccount != solana.PublicKey{} &&
		a.AmmTargetOrders != solana.PublicKey{} && a.AmmLiquidityCreator != solana.PublicKey{} && a.Caller != solana.PublicKey{} &&
		a.TxID != solana.Signature{} && a.Slot != 0 && !a.TxTime.IsZero() && !a.Timestamp.IsZero() && a.InitialLiveInfo.Ready() && a.CurrentLiveInfo.Ready()
}

// Update amm info if market pair was created in reverse order to match the order of the market.
func (a *AmmInfo) UpdateSwap(ammSwapped bool) {
	defer a.initializeCurrent()

	if !ammSwapped {
		return
	}

	fmt.Printf("[%v] Swapping amm info (before): token: %s, currency: %s ... (after): token: %s, currency: %s\n", time.Now().Format("2006-01-02 15:04:05.000"), a.TokenMintAddress, a.CurrencyAddress, a.CurrencyAddress, a.TokenMintAddress)
	a.TokenMintAddress, a.CurrencyAddress = a.CurrencyAddress, a.TokenMintAddress
	a.PoolCoinTokenAccount, a.PoolPcTokenAccount = a.PoolPcTokenAccount, a.PoolCoinTokenAccount
	a.InitialLiveInfo.PooledToken, a.InitialLiveInfo.PooledLamports = a.InitialLiveInfo.PooledLamports, a.InitialLiveInfo.PooledToken
	a.InitialLiveInfo.Price = 1.0 / a.InitialLiveInfo.Price
}

func (a *AmmInfo) initializeCurrent() {
	a.CurrentLiveInfo = a.InitialLiveInfo
}

func (a *AmmInfo) TokenAddress() solana.PublicKey {
	return a.TokenMintAddress
}

func (a *AmmInfo) CoinVault() solana.PublicKey {
	return a.PoolCoinTokenAccount
}

func (a *AmmInfo) PcVault() solana.PublicKey {
	return a.PoolPcTokenAccount
}

func DeriveAmmInfoFromMarket(minfo serum.MarketInfo) (*AmmInfo, error) {
	ainfo := &AmmInfo{}

	// === Needed for swap ===

	ammId, _, err := solana.FindProgramAddress([][]byte{
		Raydium_Liquidity_Program_V4.Bytes(),
		minfo.Market.Bytes(),
		[]byte("amm_associated_seed"),
	}, Raydium_Liquidity_Program_V4)
	if err != nil {
		return nil, fmt.Errorf("amm_associated_seed error: %v", err)
	}
	ainfo.AmmID = ammId

	ammPoolCoinTokenAccount, _, err := solana.FindProgramAddress([][]byte{
		Raydium_Liquidity_Program_V4.Bytes(),
		minfo.Market.Bytes(),
		[]byte("coin_vault_associated_seed"),
	}, Raydium_Liquidity_Program_V4)
	if err != nil {
		return nil, fmt.Errorf("coin_vault_associated_seed error: %v", err)
	}
	ainfo.PoolCoinTokenAccount = ammPoolCoinTokenAccount

	ammPoolPcTokenAccount, _, err := solana.FindProgramAddress([][]byte{
		Raydium_Liquidity_Program_V4.Bytes(),
		minfo.Market.Bytes(),
		[]byte("pc_vault_associated_seed"),
	}, Raydium_Liquidity_Program_V4)
	if err != nil {
		return nil, fmt.Errorf("pc_vault_associated_seed error: %v", err)
	}
	ainfo.PoolPcTokenAccount = ammPoolPcTokenAccount

	ammPoolTokenMint, _, err := solana.FindProgramAddress([][]byte{
		Raydium_Liquidity_Program_V4.Bytes(),
		minfo.Market.Bytes(),
		[]byte("lp_mint_associated_seed"),
	}, Raydium_Liquidity_Program_V4)
	if err != nil {
		return nil, fmt.Errorf("lp_mint_associated_seed error: %v", err)
	}
	ainfo.LPTokenAddress = ammPoolTokenMint

	ammTargetOrders, _, err := solana.FindProgramAddress([][]byte{
		Raydium_Liquidity_Program_V4.Bytes(),
		minfo.Market.Bytes(),
		[]byte("target_associated_seed"),
	}, Raydium_Liquidity_Program_V4)
	if err != nil {
		return nil, fmt.Errorf("target_associated_seed error: %v", err)
	}
	ainfo.AmmTargetOrders = ammTargetOrders

	ammOpenOrders, _, err := solana.FindProgramAddress([][]byte{
		Raydium_Liquidity_Program_V4.Bytes(),
		minfo.Market.Bytes(),
		[]byte("open_order_associated_seed"),
	}, Raydium_Liquidity_Program_V4)
	if err != nil {
		return nil, fmt.Errorf("open_order_associated_seed error: %v", err)
	}
	ainfo.AmmOpenOrders = ammOpenOrders

	// === Needed for LP token burn check ===

	ainfo.TokenMintAddress = minfo.BaseMint
	ainfo.CurrencyAddress = minfo.QuoteMint
	ainfo.Calculated = true
	ainfo.TxID = minfo.TxID
	ainfo.TxTime = minfo.TxTime

	return ainfo, nil
}

func AmmInfoFromTransaction(rpcTx *rpc.GetTransactionResult, tx *solana.Transaction, txParsedLog *json.RawMessage) (AmmInfo, error) {
	safeIndex := func(idx uint16) solana.PublicKey {
		if idx >= uint16(len(tx.Message.AccountKeys)) {
			return solana.PublicKey{}
		}
		return tx.Message.AccountKeys[idx]
	}

	ainfo := NewAmmInfo()
	found := false

	for _, instr := range tx.Message.Instructions {
		program, err := tx.Message.Program(instr.ProgramIDIndex)
		if err != nil {
			continue // Program account index out of range.
		}

		if program.String() != Raydium_Liquidity_Program_V4.String() {
			continue // Not called by serum.
		}

		if len(instr.Accounts) < 21 {
			continue // Not enough accounts for Purchase IDO instruction.
		}

		ainfo.AmmID = safeIndex(instr.Accounts[4])
		ainfo.AmmOpenOrders = safeIndex(instr.Accounts[6])
		ainfo.LPTokenAddress = safeIndex(instr.Accounts[7])
		ainfo.TokenMintAddress = safeIndex(instr.Accounts[8])
		ainfo.CurrencyAddress = safeIndex(instr.Accounts[9])
		ainfo.PoolCoinTokenAccount = safeIndex(instr.Accounts[10])
		ainfo.PoolPcTokenAccount = safeIndex(instr.Accounts[11])
		ainfo.AmmTargetOrders = safeIndex(instr.Accounts[12])
		ainfo.AmmLiquidityCreator = safeIndex(instr.Accounts[20])

		ainfo.Caller = tx.Message.AccountKeys[0] // Should be ok, but not sure.
		ainfo.TxID = tx.Signatures[0]
		found = true
		break
	}

	if !found {
		return AmmInfo{}, fmt.Errorf("no Purchase IDO instruction found")
	}

	if err := ainfo.deriveFromParsedLog(txParsedLog); err != nil {
		return AmmInfo{}, fmt.Errorf("deriveFromParsedLog error: %w", err)
	}

	ainfo.Slot = rpcTx.Slot
	ainfo.TxTime = rpcTx.BlockTime.Time()
	ainfo.Timestamp = time.Now()

	return *ainfo, nil
}

func (a *AmmInfo) deriveFromParsedLog(txParsedLog *json.RawMessage) error {
	type idoinfo struct {
		Nonce          float64 `json:"nonce"`
		OpenTime       float64 `json:"open_time"`
		InitPcAmount   float64 `json:"init_pc_amount"`
		InitCoinAmount float64 `json:"init_coin_amount"`
	}

	var ido idoinfo
	if err := json.Unmarshal(*txParsedLog, &ido); err != nil {
		return fmt.Errorf("ido log unmarshal error: %w", err)
	}

	a.InitialLiveInfo.UpdateTime = time.Unix(int64(ido.OpenTime), 0)
	a.InitialLiveInfo.PooledLamports = ido.InitPcAmount
	a.InitialLiveInfo.PooledToken = ido.InitCoinAmount
	a.InitialLiveInfo.Price = a.InitialLiveInfo.PooledToken / a.InitialLiveInfo.PooledLamports

	return nil
}
