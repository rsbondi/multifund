package wallet

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/rsbondi/multifund/rpc"
)

type BitcoinWallet struct {
	rpchost     string
	rpcport     string
	rpcuser     string
	rpcpassword string
}

func NewBitcoinWallet() *BitcoinWallet {
	cfg, err := rpc.ListConfigs()
	if err != nil {
		log.Fatal(err)
	}
	var host, user, pass string
	var port string
	if cfg.BitcoinRpcConnect != "" && cfg.BitcoinRpcUser != "" && cfg.BitcoinRpcPassword != "" {
		user = cfg.BitcoinRpcUser
		pass = cfg.BitcoinRpcPassword
		connect := strings.Split(cfg.BitcoinRpcConnect, ":")
		host = connect[0]
		port = connect[1]
	} else if cfg.BitcoinRpcPort != "" && cfg.BitcoinRpcUser != "" && cfg.BitcoinRpcPassword != "" {
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
	Txid          string  `json:"txid"`
	Vout          uint32  `json:"vout"`
	Amount        float32 `json:"amount"`
	ScriptPubKey  string  `json:"scriptPubKey"`
	RedeemScript  string  `json:"redeemScript"`
	Confirmations uint    `json:"confirmations"`
}

type empty struct{}

func makeResult(r interface{}) RpcResult {
	e := &RpcError{}
	result := RpcResult{
		Result: r,
		Error:  e,
	}
	return result
}

type ByMsat []utxo

func (a ByMsat) Len() int           { return len(a) }
func (a ByMsat) Less(i, j int) bool { return a[i].Amount < a[j].Amount }
func (a ByMsat) Swap(i, j int)      { a[i].Amount, a[j].Amount = a[j].Amount, a[i].Amount }

func (b *BitcoinWallet) Utxos(amt uint64, fee uint64) ([]UTXO, error) {
	minconf := uint(3)
	unspent := make([]utxo, 0)
	result := makeResult(&unspent)
	b.RpcPost("listunspent", []empty{}, &result)
	dust := uint64(1000)
	candidates := make([]utxo, 0)
	for _, u := range unspent {
		sats := Satoshis(u.Amount)
		if sats == amt+fee && u.Confirmations > minconf {
			txid, err := hex.DecodeString(u.Txid)
			if err != nil {
				log.Printf("unable to decode txid %s\n", err)

			}
			h, _ := chainhash.NewHash(txid)
			o := wire.NewOutPoint(h, u.Vout)
			utxos := []UTXO{UTXO{Satoshis(u.Amount), *o}}
			return utxos, nil
		}
		if sats > amt+fee+dust && u.Confirmations > minconf {
			candidates = append(candidates, u)
		}
	}
	if len(candidates) == 0 {
		return nil, errors.New("no utxo candidates available") // TODO: try multiple
	}

	sort.Sort(ByMsat(candidates))
	c := candidates[0]
	txid, err := hex.DecodeString(c.Txid)
	if err != nil {
		log.Printf("unable to decode txid %s\n", err)

	}
	h, err := chainhash.NewHash(txid)
	if err != nil {
		log.Printf("unable to create hash from txid %s\n", err)

	}
	o := wire.NewOutPoint(h, c.Vout)

	utxos := []UTXO{UTXO{Satoshis(c.Amount), *o}}
	return utxos, nil
}

type EstimateSmartFeeResult struct {
	Feerate float32 `json:"feerate"`
}

func (b *BitcoinWallet) EstimateSmartFee(target uint) RpcResult {
	fee := EstimateSmartFeeResult{}
	result := makeResult(&fee)
	b.RpcPost("estimatesmartfee", []uint{target}, &result)

	return result
}

func (b *BitcoinWallet) ChangeAddress() string {
	addr := ""
	result := makeResult(&addr)
	b.RpcPost("getnewaddress", []string{"", "bech32"}, &result)
	return addr
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
	url := fmt.Sprintf("http://%s:%s", b.rpchost, b.rpcport)
	rpcCall := &RpcCall{
		Id:      time.Now().Unix(),
		Method:  method,
		JsonRpc: "2.0",
		Params:  params,
	}
	jsoncall, err := json.Marshal(rpcCall)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsoncall))
	basic := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", b.rpcuser, b.rpcpassword))))
	req.Header.Set("Authorization", basic)
	client := &http.Client{Timeout: time.Second * 10}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(result)

	return err
}

func (b *BitcoinWallet) Sign(tx *Transaction, outputs map[string]*Outputs) {

}

func SendTx(rawtx string) {

}
