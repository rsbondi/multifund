package wallet

import (
	"github.com/btcsuite/btcd/wire"
	"github.com/niftynei/glightning/glightning"
)

type InternalWallet struct {
	lightning *glightning.Lightning
}

func NewInternalWallet(l *glightning.Lightning) *InternalWallet {
	return &InternalWallet{
		lightning: l,
	}
}

func (i *InternalWallet) Utxos(amt uint64) []wire.OutPoint {
	return nil
}

func (i *InternalWallet) ChangeAddress() string {
	return ""
}
