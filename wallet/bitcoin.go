package wallet

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/rsbondi/multifund/rpc"
)

type BitcoinWallet struct {
	rpchost     string
	rpcport     int
	rpcuser     string
	rpcpassword string
}

func Satoshis(btc float32) uint64 {
	return uint64(btc * float32(100000000))
}

func NewBitcoinWallet() *BitcoinWallet {
	cfg, err := rpc.ListConfigs()
	if err != nil {
		log.Fatal(err)
	}
	var host, user, pass string
	var port int
	if cfg.BitcoinRpcConnect != "" && cfg.BitcoinRpcUser != "" && cfg.BitcoinRpcPassword != "" {
		user = cfg.BitcoinRpcUser
		pass = cfg.BitcoinRpcPassword
		connect := strings.Split(cfg.BitcoinRpcConnect, ":")
		host = connect[0]
		port, err = strconv.Atoi(connect[1])
		if err != nil {
			log.Fatal(err)
		}
	} else if cfg.BitcoinRpcPort != 0 && cfg.BitcoinRpcUser != "" && cfg.BitcoinRpcPassword != "" {
		user = cfg.BitcoinRpcUser
		pass = cfg.BitcoinRpcPassword
		host = "127.0.0.1"
		port = cfg.BitcoinRpcPort
	} else {
		// TODO: get from ~/.bitcoin/bitcoin.conf
	}

	return &BitcoinWallet{
		rpchost:     host,
		rpcport:     port,
		rpcuser:     user,
		rpcpassword: pass,
	}
}

type utxo struct {
	Txid          []byte  `json:"txid"`
	Vout          uint32  `json:"vout"`
	Amount        float32 `json:"amount"`
	ScriptPubKey  []byte  `json:"scriptPubKey"`
	RedeemScript  []byte  `json:"redeemScript"`
	Confirmations uint    `json:"confirmations"`
}

type empty struct{}

func makeResult(r interface{}) *RpcResult {
	e := &RpcError{}
	result := &RpcResult{
		Result: r,
		Error:  e,
	}
	return result
}

type ByMsat []utxo

func (a ByMsat) Len() int           { return len(a) }
func (a ByMsat) Less(i, j int) bool { return a[i].Amount < a[j].Amount }
func (a ByMsat) Swap(i, j int)      { a[i].Amount, a[j].Amount = a[j].Amount, a[i].Amount }

func (b *BitcoinWallet) Utxos(amt uint64) ([]wire.OutPoint, error) {
	rate := b.EstimateSmartFee(100)
	if rate.Error != nil {
		return nil, errors.New(rate.Error.Message)
	}
	minconf := uint(3)
	unspent := make([]utxo, 0)
	result := makeResult(&unspent)
	b.RpcPost("listunspent", []empty{}, &result)
	fee := Satoshis(rate.Result.(EstimateSmartFeeResult).Feerate / 1000.0) // TODO: calculate kb
	dust := uint64(1000)
	candidates := make([]utxo, 0)
	for _, u := range unspent {
		sats := Satoshis(u.Amount)
		if sats == amt+fee && u.Confirmations > minconf {
			h, _ := chainhash.NewHash(u.Txid)
			o := wire.NewOutPoint(h, u.Vout)
			return []wire.OutPoint{*o}, nil
		}
		if sats > amt+fee+dust && u.Confirmations > minconf {
			candidates = append(candidates, u)
		}
	}
	if len(candidates) == 0 {
		return nil, errors.New("no utxo candidates available") // TODO: try multiple
	}
	sort.Sort(ByMsat(candidates))
	u := candidates[0]
	h, _ := chainhash.NewHash(u.Txid)
	o := wire.NewOutPoint(h, u.Vout)

	return []wire.OutPoint{*o}, nil
}

type EstimateSmartFeeResult struct {
	Feerate float32 `json:"feerate"`
}

func (b *BitcoinWallet) EstimateSmartFee(target uint) *RpcResult {
	fee := EstimateSmartFeeResult{}
	result := makeResult(&fee)
	b.RpcPost("estimatesmartfee", []uint{target}, &result)
	return result
}

func (b *BitcoinWallet) ChangeAddress() string {
	return ""
}

type RpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type RpcResult struct {
	Result interface{} `json:"result"`
	Error  *RpcError   `json:"error"`
}

type RpcCall struct {
	Id      int64       `json:"id"`
	Method  string      `json:"method"`
	JsonRpc string      `json:"jsonrpc"`
	Params  interface{} `json:"params"`
}

func (b *BitcoinWallet) RpcPost(method string, params interface{}, result interface{}) error {
	url := fmt.Sprintf("%s:%d", b.rpchost, b.rpcport)
	rpcCall := &RpcCall{
		Id:      time.Now().Unix(),
		Method:  method,
		JsonRpc: "2.0",
		Params:  params,
	}
	jsoncall, err := json.Marshal(rpcCall)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsoncall))
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", b.rpcuser, b.rpcpassword)))))
	req.Header.Add("Content-type", "application/json")
	client := &http.Client{Timeout: time.Second * 10}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	rpcres := &RpcResult{
		Result: result,
		Error:  &RpcError{},
	}

	err = json.NewDecoder(res.Body).Decode(rpcres)
	return err
}

func SendTx(rawtx string) {

}
