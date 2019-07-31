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

	// Utxos will provide utxos(wire.OutPoint) for the wallet implementation based on the amount
	// amt is the amount of the transaction used to determine what utxos to use to cover the amount plus fees
	Utxos(amt uint64, fee uint64) ([]UTXO, error)

	// ChangeAddress provides where to send the change
	ChangeAddress() string

	// Sign uses the wallet implementation to provide signatures so a transaction can be broadcast
	// tx is the transaction to be sighned
	// utxos provides the transaction inputs that need signing, from this it should be able to locate
	//   the private keys
	Sign(tx *Transaction, utxos []UTXO)
}

type UTXO struct {
	Amount  uint64
	Address string
	wire.OutPoint
}

func reverseBytes(b []byte) []byte {
	newbytes := make([]byte, len(b))
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		newbytes[i], newbytes[j] = b[j], b[i]
	}
	return newbytes
}

func Satoshis(btc float64) uint64 {
	return uint64(btc * float64(100000000))
}
