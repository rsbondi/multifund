package wallet

import (
	"github.com/btcsuite/btcd/wire"
)

const (
	WALLET_BITCOIN int = iota
	WALLET_EXTERNAL
	WALLET_INTERNAL
)

type Wallet interface {

	// GetUtxos will provide utxos(wire.OutPoint) for the wallet implementation based on the amount
	// amt is the amount of the transaction used to determine what utxos to use to cover the amount plus fees
	Utxos(amt uint64) []wire.OutPoint

	// GetChangeAddress provides where to send the change
	ChangeAddress() string
}