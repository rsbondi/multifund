package main

import (
	"bytes"
	"errors"
	"log"

	"github.com/btcsuite/btcd/wire"
	"github.com/niftynei/glightning/jrpc2"
	"github.com/rsbondi/multifund/rpc"
	"github.com/rsbondi/multifund/wallet"
)

const FundMultiDescription = `Use external wallet funding feature to build a transaction to fund multiple channels
{channels} is an array of object{"id" string, "satoshi" int, "announce" bool}`

type MultiChannel struct {
	Channels []rpc.FundChannelStartRequest
}

func (m *MultiChannel) Call() (jrpc2.Result, error) {
	return createMulti(&m.Channels)
}

func (f *MultiChannel) Name() string {
	return "fund_multi"
}

func (f *MultiChannel) New() interface{} {
	return &MultiChannel{}
}

type ConnectAndFundChannelRequest struct {
	Id       string  `json:"id"`
	Host     string  `json:"host,omitempty"`
	Port     float64 `json:"port,omitempty"`
	Amount   float64 `json:"satoshi"`
	FeeRate  string  `json:"feerate,omitempty"`
	Announce bool    `json:"announce"`
}

type MultiChannelWithConnect struct {
	Channels []ConnectAndFundChannelRequest
}

var wally wallet.Wallet

func (m *MultiChannelWithConnect) Call() (jrpc2.Result, error) {
	return connectAndCreateMulti(&m.Channels)
}

func (f *MultiChannelWithConnect) Name() string {
	return "connect_fund_multi"
}

func (f *MultiChannelWithConnect) New() interface{} {
	return &MultiChannelWithConnect{}
}

func connectAndCreateMulti(chans *[]ConnectAndFundChannelRequest) (jrpc2.Result, error) {
	createChans := make([]rpc.FundChannelStartRequest, 0)
	for _, c := range *chans {
		_, err := lightning.Connect(c.Id, c.Host, uint(c.Port))
		if err != nil {
			return nil, err
		}
		newone := rpc.FundChannelStartRequest{
			Id:       c.Id,
			Amount:   c.Amount,
			FeeRate:  c.FeeRate,
			Announce: c.Announce,
		}
		createChans = append(createChans, newone)
	}

	return createMulti(&createChans)
}

type FundingInfo struct {
	Outputs    map[string]*wallet.Outputs
	Recipients []*wallet.TxRecipient
	Utxos      []wallet.UTXO
}

// GetChannelAddresses provides funding information for creating a transaction
//   the transaction can be created here or by sending the info to an external server
//   this opens the potential for a multi party channel opening, or use of an external
//   manual wallet signing
func GetChannelAddresses(chans *[]rpc.FundChannelStartRequest) (*FundingInfo, error) {
	var recipients = make([]*wallet.TxRecipient, 0)
	outputs := make(map[string]*wallet.Outputs, 0)
	outamt := uint64(0)
	rate := bitcoin.EstimateSmartFee(100)
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

	if wally == nil {
		switch wallettype {
		case wallet.WALLET_BITCOIN:
			wally = bitcoin
		case wallet.WALLET_INTERNAL:
			wally = InternalWallet()
		}
	}

	change := wally.ChangeAddress()
	utxos, err := wally.Utxos(outamt, fee)
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
		result, err := rpc.FundChannelStart(c.Id, c.Amount)
		if err != nil {
			return nil, err
		}
		amt := int64(c.Amount) // difference in wire and glightning
		outputs[c.Id] = &wallet.Outputs{Vout: i, Amount: amt, Address: result.FundingAddress}
		recipamt += amt
		recipients = append(recipients, &wallet.TxRecipient{Address: result.FundingAddress, Amount: amt})
	}

	if utxoamt-fee > wallet.DUST_LIMIT { // no change if dust, save on tx fee
		recipients = append(recipients, &wallet.TxRecipient{Address: change, Amount: int64(utxoamt-fee) - recipamt})
		// recalculate fee, for more accureate change amount
		vsize := wallet.InputFeeSats(utxos, bitcoinNet) + wallet.OutputFeeSats(recipients, bitcoinNet) + 11
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

func createMulti(chans *[]rpc.FundChannelStartRequest) (jrpc2.Result, error) {
	info, err := GetChannelAddresses(chans)
	if err != nil {
		cancelMulti(info.Outputs)
		return nil, err
	}

	tx, err := wallet.CreateTransaction(info.Recipients, info.Utxos, bitcoinNet)
	if err != nil {
		return nil, err
	}

	wally.Sign(&tx, info.Utxos)
	wtx := wire.NewMsgTx(2)
	r := bytes.NewReader(tx.Signed)
	wtx.Deserialize(r)
	tx.TxId = wtx.TxHash().String()

	channels := make([]*rpc.FundChannelCompleteResponse, 0)
	for k, o := range info.Outputs {
		cid, err := rpc.FundChannelComplete(k, tx.TxId, o.Vout)
		if err != nil {
			closeMulti(info.Outputs)
			return nil, err
		}
		channels = append(channels, cid)
	}

	txid, err := bitcoin.SendTx(tx.String())
	if err != nil {
		closeMulti(info.Outputs)
		return nil, err
	}

	return struct {
		Tx       string                             `json:"tx"`
		Txid     string                             `json:"txid"`
		Channels []*rpc.FundChannelCompleteResponse `json:"channels"`
	}{
		tx.String(),
		txid,
		channels,
	}, nil
}

func cancelMulti(outputs map[string]*wallet.Outputs) {
	for k, _ := range outputs {
		_, err := rpc.FundChannelCancel(k)
		if err != nil {
			log.Printf("fundchannel_cancel error: %s", err.Error())
		}
	}
}

func closeMulti(outputs map[string]*wallet.Outputs) {
	for k, _ := range outputs {
		_, err := lightning.CloseNormal(k)
		if err != nil {
			log.Printf("channel close error: %s", err.Error())
		}
	}
}
