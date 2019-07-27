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
	Utxos(amt uint64, fee uint64) ([]UTXO, error)

	// GetChangeAddress provides where to send the change
	ChangeAddress() string
}

type UTXO struct {
	Amount uint64
	wire.OutPoint
}

func Satoshis(btc float32) uint64 {
	return uint64(btc * float32(100000000))
}
