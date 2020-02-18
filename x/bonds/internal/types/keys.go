package types

const (
	// ModuleName is the name of this module
	ModuleName = "bonds"

	// StoreKey is the default store key for this module
	StoreKey = ModuleName

	// BondsMintBurnAccount the root string for the bonds mint burn account address
	BondsMintBurnAccount = "bonds_mint_burn_account"

	// BatchesIntermediaryAccount the root string for the batches account address
	BatchesIntermediaryAccount = "batches_intermediary_account"

	// QuerierRoute is the querier route for this module's store.
	QuerierRoute = ModuleName

	// RouterKey is the message route for this module
	RouterKey = ModuleName
)

// Bonds and batches are stored as follow:
//
// - Bonds: 0x00<bond_token_bytes>
// - Batches: 0x01<bond_token_bytes>
// - Last batches: 0x01<bond_token_bytes>
var (
	BondsKeyPrefix       = []byte{0x00} // key for bonds
	BatchesKeyPrefix     = []byte{0x01} // key for batches
	LastBatchesKeyPrefix = []byte{0x02} // key for last batches
)

func GetBondKey(token string) []byte {
	return append(BondsKeyPrefix, []byte(token)...)
}

func GetBatchKey(token string) []byte {
	return append(BatchesKeyPrefix, []byte(token)...)
}

func GetLastBatchKey(token string) []byte {
	return append(LastBatchesKeyPrefix, []byte(token)...)
}
