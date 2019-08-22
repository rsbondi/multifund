package funder

import (
	"bytes"
	"encoding/hex"
	"errors"
	"log"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/niftynei/glightning/glightning"
	"github.com/rsbondi/multifund/wallet"
)

type Funder struct {
	Lightning      *glightning.Lightning
	Wallettype     int
	Bitcoin        *wallet.BitcoinWallet // we always use this at least for broadcasting the tx
	Internal       wallet.Wallet
	Wally          wallet.Wallet
	BitcoinNet     *chaincfg.Params
	Lightningdir   string
	internalWallet *wallet.InternalWallet
}

type FundingInfo struct {
	Outputs    map[string]*wallet.Outputs
	Recipients []*wallet.TxRecipient
	Utxos      []wallet.UTXO
}

func (f *Funder) InternalWallet() wallet.Wallet {
	if f.internalWallet == nil {
		f.internalWallet = wallet.NewInternalWallet(f.Lightning, f.BitcoinNet, f.Lightningdir)
	}
	return f.internalWallet
}

// GetChannelAddresses provides funding information for creating a transaction
//   the transaction can be created here or by sending the info to an external server
//   this opens the potential for a multi party channel opening, or use of an external
//   manual wallet signing
// returns a FundingInfo struct with state, recipients and utxos
func (f *Funder) GetChannelAddresses(chans *[]glightning.FundChannelStart) (*FundingInfo, error) {
	var recipients = make([]*wallet.TxRecipient, 0)
	outputs := make(map[string]*wallet.Outputs, 0)
	outamt := uint64(0)
	rate := f.Bitcoin.EstimateSmartFee(100)
	// fee calc, we know the output rate, type is known before we create the addresses
	//   43 vbytes per channel
	// we don't know how many utxos, and wee need some starting point to fetch them
	//   so we have a chicken/egg scenario
	//   this could be way off if a bunch of small utxos, but we have the dust buffer
	//   and we may not use change if we are within the dust buffer
	//   this may need further consideration
	bytesEstimate := uint64(160 + 43*len(*chans)) // this may change if we need mor utxos
	feerate := rate.Result.(*wallet.EstimateSmartFeeResult).Feerate / 1000.0

	var fee uint64
	if feerate == 0.0 {
		log.Println("unable to estimate fee rate, using default")
		fee = 2 * bytesEstimate
	} else {
		fee = wallet.Satoshis(feerate) * bytesEstimate
	}

	recipamt := int64(0)
	for _, c := range *chans {
		outamt += uint64(c.Amount)
	}

	if f.Wally == nil {
		switch f.Wallettype {
		case wallet.WALLET_BITCOIN:
			f.Wally = f.Bitcoin
		case wallet.WALLET_INTERNAL:
			f.Wally = f.InternalWallet()
		}
	}

	change := f.Wally.ChangeAddress()
	utxos, err := f.Wally.Utxos(outamt, fee)
	if err != nil {
		return nil, err
	}
	utxoamt := uint64(0)
	for _, u := range utxos {
		utxoamt += u.Amount
	}

	if outamt > utxoamt+fee {
		return nil, errors.New("Insufficient funds, Need more coin")
	}

	for i, c := range *chans {
		result, err := f.Lightning.StartFundChannel(c.Id, c.Amount, c.Announce, nil)
		if err != nil {
			log.Printf("fund start error: %s", err.Error())
			return nil, err
		}
		addr, err := btcutil.DecodeAddress(result, f.BitcoinNet)
		addr.ScriptAddress()

		amt := int64(c.Amount) // difference in wire and glightning
		outputs[c.Id] = &wallet.Outputs{Vout: uint16(i), Amount: amt, Script: addr.ScriptAddress()}
		recipamt += amt
		recipients = append(recipients, &wallet.TxRecipient{Address: result, Amount: amt})
	}

	if utxoamt-fee > wallet.DUST_LIMIT { // no change if dust, save on tx fee
		recipients = append(recipients, &wallet.TxRecipient{Address: change, Amount: int64(utxoamt-fee) - recipamt})
		// recalculate fee, for more accureate change amount
		vsize := wallet.InputFeeSats(utxos, f.BitcoinNet) + wallet.OutputFeeSats(recipients, f.BitcoinNet) + 11
		if feerate == 0.0 {
			fee = 2 * vsize
		} else {
			fee = wallet.Satoshis(feerate) * vsize
		}
		recipients[len(recipients)-1].Amount = int64(utxoamt-fee) - recipamt
	}
	fundinfo := &FundingInfo{
		Outputs:    outputs,
		Recipients: recipients,
		Utxos:      utxos,
	}
	return fundinfo, nil
}

func (f *Funder) CompleteChannels(tx wallet.Transaction, outputs map[string]*wallet.Outputs) ([]string, error) {
	channels := make([]string, 0)
	wtx := wire.NewMsgTx(2)
	r := bytes.NewReader(tx.Signed)
	wtx.Deserialize(r)

	for k, o := range outputs {
		vout := -1
		for v, txout := range wtx.TxOut {
			log.Printf("finding output index: %v %v %d", txout.PkScript[2:], o.Script, v)
			if hex.EncodeToString(txout.PkScript[2:]) == hex.EncodeToString(o.Script) {
				if o.Amount != txout.Value {
					return nil, errors.New("Can not find output in transaction")
				}
				vout = v
				break
			}
		}
		if vout == -1 {
			return nil, errors.New("Can not find output in transaction")
		}
		cid, err := f.Lightning.CompleteFundChannel(k, tx.TxId, uint16(vout))
		if err != nil {
			return nil, err
		}
		channels = append(channels, cid)
	}
	return channels, nil
}
