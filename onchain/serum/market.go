package serum

import (
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// Serum's OpenBookDex program ID.
var OpenBookDex solana.PublicKey = solana.MustPublicKeyFromBase58("srmqPvymJeFKQ4zGQed1GFppgkRHL9kaELCbyksJtPX")

// This is how to calculate addresses by yourself: https://github.com/project-serum/serum-dex/blob/master/dex/crank/src/lib.rs#L1258
//
// Serum Initialize Market: https://solscan.io/tx/TuSSLHokzkxVFJ5PmP2FBkW3W9UUMJf6hBbrTRGBBgpSALQfwKvbWVYR2bMvW2dKis1U4oqJLumKpbyfGj6C6Z9
// https://explorer.solana.com/tx/TuSSLHokzkxVFJ5PmP2FBkW3W9UUMJf6hBbrTRGBBgpSALQfwKvbWVYR2bMvW2dKis1U4oqJLumKpbyfGj6C6Z9

// Transaction Version: legacy

// programID -> OpenBook
// account1 -> Market -> SerumMarket
// account2 -> RequestQueue (unused)
// account3 -> EventQueue -> SerumEventQueue
// account4 -> Bids -> SerumBids
// account5 -> Asks -> SerumAsks
// account6 -> BaseVault -> SerumCoinVaultAccount
// account7 -> QuoteVault -> SerumPcVaultAccount
// account8 -> BaseMint (TokenAddress)
// account9 -> QuoteMint (CurrencyAddress - chcemy tylko WSOL, więc So11111111111111111111111111111111111111112)
// account10 -> Sysvar: Rent
type MarketInfo struct {
	// Initialize Market Instruction Data
	ProgramID    solana.PublicKey // always OpenBook v3
	Market       solana.PublicKey // serum market address
	RequestQueue solana.PublicKey // serum request queue address (unused for swaps)
	EventQueue   solana.PublicKey // serum event queue address
	Bids         solana.PublicKey // serum bids address
	Asks         solana.PublicKey // serum asks address
	BaseMint     solana.PublicKey // base mint address (Token Address)
	QuoteMint    solana.PublicKey // quote mint address (Currency Address)
	BaseVault    solana.PublicKey // base vault address (Token Account)
	QuoteVault   solana.PublicKey // quote vault address (Currency Account)
	Sysvar       solana.PublicKey // SYSVAR_CLOCK_PUBKEY

	// Initialize Market Instruction Metadata
	Caller    solana.PublicKey // Caller wallet address
	TxID      solana.Signature // Transaction ID
	Slot      uint64           // Chain Slot
	TxTime    time.Time        // Timestamp of transaction in blockchain
	Timestamp time.Time        // Timestamp of transaction discovery
	Swapped   bool             // Whether the pair was created in reverse order.

	// Initialize Market Instruction Extra
	VaultSigner solana.PublicKey // Vault signer; this value is provided by separate RPC call.
}

// Initialize market info with hardcoded values.
// The rest of the values will be filled in by the Update method.
func NewMarketInfo() *MarketInfo {
	return &MarketInfo{
		ProgramID: OpenBookDex,
		Sysvar:    solana.SysVarClockPubkey,
	}
}

// Ready returns true if all the required fields are filled in. This should be true after Update.
// MarketInfo and raydium.AmmInfo both has to be ready before the swap can be executed.
func (m *MarketInfo) Ready() bool {
	return m.Market != solana.PublicKey{} && m.EventQueue != solana.PublicKey{} && m.Bids != solana.PublicKey{} &&
		m.Asks != solana.PublicKey{} && m.BaseMint != solana.PublicKey{} && m.QuoteMint != solana.PublicKey{} &&
		m.BaseVault != solana.PublicKey{} && m.QuoteVault != solana.PublicKey{} && m.Caller != solana.PublicKey{} &&
		m.TxID != solana.Signature{} && m.Slot != 0 && m.TxTime != time.Time{} && m.Timestamp != time.Time{} &&
		m.VaultSigner != solana.PublicKey{}
}

// TokenAddress returns the token address of the market.
func (m *MarketInfo) TokenAddress() solana.PublicKey {
	return m.BaseMint
}

// Market's token vault.
func (m *MarketInfo) CoinVault() solana.PublicKey {
	return m.BaseVault
}

// Market's currency vault.
func (m *MarketInfo) PcVault() solana.PublicKey {
	return m.QuoteVault
}

// serum market initialization input data:
// 34 bytes, 5 args + 5 bytes unknown
//
//	(0, 34) => MarketInstruction::InitializeMarket({
//		let data_array = array_ref![data, 0, 34];
//		let fields = array_refs![data_array, 8, 8, 2, 8, 8];
//		InitializeMarketInstruction {
//			coin_lot_size: u64::from_le_bytes(*fields.0),
//			pc_lot_size: u64::from_le_bytes(*fields.1),
//			fee_rate_bps: u16::from_le_bytes(*fields.2),
//			vault_signer_nonce: u64::from_le_bytes(*fields.3),
//			pc_dust_threshold: u64::from_le_bytes(*fields.4),
//		}
//	}),
//
// const vaultSigner = await PublicKey.createProgramAddress(
//
//		[
//		  this.address.toBuffer(),
//		  this._decoded.vaultSignerNonce.toArrayLike(Buffer, 'le', 8),
//		],
//		this._programId,
//	  );
//
// It seems somehow it doesnt match above. VaultSignerNonce starts at 23th because of additional 5 bytes (method id??)
func MarketInfoFromTransaction(rpcTx *rpc.GetTransactionResult, tx *solana.Transaction) (MarketInfo, error) {
	minfo := NewMarketInfo()
	for _, instr := range tx.Message.Instructions {
		program, err := tx.Message.Program(instr.ProgramIDIndex)
		if err != nil {
			continue // Program account index out of range.
		}

		if program.String() != OpenBookDex.String() {
			continue // Not called by serum.
		}

		if len(instr.Accounts) < 10 {
			continue // Not enough accounts for InitializeMarket instruction.
		}

		const BaseMintIndex = 7
		const QuoteMinIndex = 8
		const SysVarRentIndex = 9

		safeIndex := func(idx uint16) solana.PublicKey {
			if idx >= uint16(len(tx.Message.AccountKeys)) {
				return solana.PublicKey{}
			}
			return tx.Message.AccountKeys[idx]
		}

		if safeIndex(instr.Accounts[QuoteMinIndex]) != solana.WrappedSol && safeIndex(instr.Accounts[BaseMintIndex]) != solana.WrappedSol {
			return MarketInfo{}, fmt.Errorf("found serum market, but not with SOL currency")
		}

		// This is probably InitializeMarket instruction.
		minfo.Market = safeIndex(instr.Accounts[0])
		minfo.EventQueue = safeIndex(instr.Accounts[2])
		minfo.Bids = safeIndex(instr.Accounts[3])
		minfo.Asks = safeIndex(instr.Accounts[4])
		minfo.BaseVault = safeIndex(instr.Accounts[5])
		minfo.QuoteVault = safeIndex(instr.Accounts[6])
		minfo.BaseMint = safeIndex(instr.Accounts[7])
		minfo.QuoteMint = safeIndex(instr.Accounts[8])

		// Swap if pair created in reverse order.
		if minfo.BaseMint == solana.WrappedSol {
			minfo.BaseVault, minfo.QuoteVault = minfo.QuoteVault, minfo.BaseVault
			minfo.BaseMint, minfo.QuoteMint = minfo.QuoteMint, minfo.BaseMint
			minfo.Swapped = true
		}

		minfo.Caller = tx.Message.AccountKeys[0] // Should be ok, but not sure.
		minfo.TxID = tx.Signatures[0]

		vaultSignerNonce := instr.Data[23:31]
		vaultsigner, err := solana.CreateProgramAddress(
			[][]byte{
				minfo.Market.Bytes()[:],
				vaultSignerNonce,
			}, OpenBookDex)

		if err != nil {
			return MarketInfo{}, fmt.Errorf("Error creating vault signer: %w", err)
		}

		minfo.VaultSigner = vaultsigner

		minfo.Slot = rpcTx.Slot
		minfo.TxTime = rpcTx.BlockTime.Time()
		minfo.Timestamp = time.Now()

		if !minfo.Ready() {
			return MarketInfo{}, fmt.Errorf("market info not ready")
		}

		return *minfo, nil
	}

	return MarketInfo{}, fmt.Errorf("no InitializeMarket instruction found")
}
