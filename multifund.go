package main

import (
	"fmt"
	"log"
	"os"

	"github.com/niftynei/glightning/glightning"
	"github.com/rsbondi/multifund/rpc"
	"github.com/rsbondi/multifund/wallet"
)

const VERSION = "0.0.1"

var plugin *glightning.Plugin
var lightning *glightning.Lightning
var wallettype int
var bitcoin *wallet.BitcoinWallet // we always use this at least for broadcasting the tx

type Outputs struct {
	Vout    int    `json:"vout"`
	Amount  int64  `json:"amount"`
	Address string `json:"address"`
}

var outputs map[string]*Outputs // hold node id to the vout position in the funding tx

func main() {
	plugin = glightning.NewPlugin(onInit)
	lightning = glightning.NewLightning()
	rpc.Init(lightning)

	registerOptions(plugin)
	registerMethods(plugin)

	err := plugin.Start(os.Stdin, os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}

func onInit(plugin *glightning.Plugin, options map[string]string, config *glightning.Config) {
	log.Printf("versiion: " + VERSION + " initialized")
	options["rpc-file"] = fmt.Sprintf("%s/%s", config.LightningDir, config.RpcFile)
	switch options["multi-wallet"] {
	case "bitcoin":
		wallettype = wallet.WALLET_BITCOIN
	case "external":
		wallettype = wallet.WALLET_EXTERNAL
	default:
		wallettype = wallet.WALLET_INTERNAL
	}
	lightning.StartUp(config.RpcFile, config.LightningDir)

	bitcoin = wallet.NewBitcoinWallet()
}

func registerOptions(p *glightning.Plugin) {
	p.RegisterOption(glightning.NewOption("multi-wallet", "Wallet to use for multi-channel open - internal, bitcoin or external", "internal"))
}

// fund_multi {"id":sats, "id2":sats, ..."idn":sats}
func registerMethods(p *glightning.Plugin) {
	multi := glightning.NewRpcMethod(&MultiChannel{}, `Open multiple channels in single transaction`)
	multi.LongDesc = FundMultiDescription
	multi.Usage = "utxos ids"
	p.RegisterMethod(multi)

	multic := glightning.NewRpcMethod(&MultiChannelComplete{}, `Finalizes multiple channels from single transaction`)
	multic.LongDesc = FundMultiCompleteDescription
	multic.Usage = "transactions"
	p.RegisterMethod(multic)

}

/*

	It seems that bitcoin rpc info is available when in ~/.lightning/config
	it does not seem to be available if these are missing, but seems to work with default ~/.bitcoin/bitcoin.conf
	so I can read from listconfigs or read from file
	I can use this at least for broadcasting
	can maybe use as an option
		--multi-wallet=[bitcoin, internal, external]
			bitcoin - uses bitcoin core wallet for funding
			internal - uses clightning internal wallet
			external - creates the raw tx, sign externally and call plugin's sendrawtx rpc command to connect to core to broadcast


	what I want to do here
	gather information about who to open channels with (id@host:port sats) ( utxos(txid vout), private keys??? ) change addresses
		option 1 would be to use the internal wallet, need to dump the keys for signing - I can't get the utility to work so this is out for now
			*call `listfunds` to get utxos result.outputs
			    "outputs": [
					{
						"txid": "13767cdbcab321f978ff658d0815be883d812d1434657c0857f502f2e4fbb608",
						"output": 1,
						"value": 75153666,
						"amount_msat": "75153666000msat",
						"address": "bcrt1qwvp8fktkxp07v0fp9jyqe7yl6rcgyu585a7pzr",
						"status": "confirmed"
					},
					{
						"txid": "d843969bffd27a27cf627966db021e6338a0f566b1869f1af4e6bb1c51729e60",
						"output": 1,
						"value": 21751329,
						"amount_msat": "21751329000msat",
						"address": "bcrt1qq98g08yqr7nynz3dqdvatppg8e9vawz5hc2n9v",
						"status": "confirmed"
					}
				]
			* do some utxo selection
			* build inputs from here
			* loop through desired peers and call fundchannel_start to get addresses
			* add these addresses to the bitcoin transaction, tracking the index to peer id
			* create the usnigned transaction

		option 2 would be to provide info on bitcoin core wallet and use it - not flexible, but may be a good starting point

		option 3 just build the transaction, let the sign externally and provide a method to broadcast and get the txid for fundchannel_complete

			this is probably the best option but less fluid than the others
	if not connected connect
	once all peers are connected
	check for active channels already with peer, error
	call fundchannel_open for each channel to get addresses
	build tx
	once broadcast, call fundchannel_complete
	broadcast

	rpc call - let's start assuming connection, figure out how to auto connect later?
	  fund_multi {"id":sats, "id2":sats, ..."idn":sats}


*/
