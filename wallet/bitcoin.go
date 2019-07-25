package wallet

import (
	"github.com/btcsuite/btcd/wire"
)

type BitcoinWallet struct {
	rpchost     string
	rpcport     int
	rpcuser     string
	rpcpassword string
}

func NewBitcoinWallet(host string, port int, user, pass string) *BitcoinWallet {
	return &BitcoinWallet{
		rpchost:     host,
		rpcport:     port,
		rpcuser:     user,
		rpcpassword: pass,
	}
}

func (i *BitcoinWallet) Utxos(amt uint64) []wire.OutPoint {
	return nil
}

func (i *BitcoinWallet) ChangeAddress() string {
	return ""
}

func SendTx(rawtx string) {

}
