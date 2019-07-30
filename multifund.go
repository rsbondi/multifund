package main

import (
	"fmt"
	"log"
	"os"

	"github.com/niftynei/glightning/glightning"
	"github.com/rsbondi/multifund/rpc"
	"github.com/rsbondi/multifund/wallet"
)

const VERSION = "0.0.1-WIP"

var plugin *glightning.Plugin
var lightning *glightning.Lightning
var wallettype int
var bitcoin *wallet.BitcoinWallet // we always use this at least for broadcasting the tx

// TODO: not sure if to keep this global and restrict to one call at a time,
//   could check for zero length to limit
//   the other option would be to keep it local and as a return value
var outputs map[string]*wallet.Outputs // hold node id to the vout position in the funding tx

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

// fund_multi [{"id":"0265b6...", "satoshi": 20000, "announce":true}, {id, satoshi, announce}...]
func registerMethods(p *glightning.Plugin) {
	multi := glightning.NewRpcMethod(&MultiChannel{}, `Open multiple channels in single transaction`)
	multi.LongDesc = FundMultiDescription
	multi.Usage = "channels"
	p.RegisterMethod(multi)

	multic := glightning.NewRpcMethod(&MultiChannelComplete{}, `Finalizes multiple channels from single transaction`)
	multic.LongDesc = FundMultiCompleteDescription
	multic.Usage = "transactions"
	p.RegisterMethod(multic)

}
