WIP - Not ready for prime time

### what

This is the start of a multi channel funding plugin for c-lightning

### why

New release allows opening of multiple channels with a single transaction.
This is an attempt to make it less painful

### status

Currently there are 2 new RPC commands

`fund_multi` and `connect_fund_multi`

`fund_multi [{"id": "02fc...", "satoshi": 20000, "announce", true}, {...}, ...]`

`connect_fund_multi` adds `"host"` and `"port"` parameters to the above

Using bitcoin core node as the wallet type works seems to be working

You must launch lightningd with the `bitcoin-xxx` either in the config file or command line.  It will eventually read them from the `bitcoin.conf` file.  The bitcoin core node is used for broadcasting transactions so it must be accessible even if you use clightning internal wallet.

TODO:
* Internal wallet working but must be bech32 inputs being spent
    * need to check input type and sign appropriately
* Read from  `bitcoin.conf`
* Implement wallet option, currently hard coded
* Read lightning config for internal wallet, currently hard coded to default location

[demo video](https://www.youtube.com/watch?v=exDYLpTncng&feature=youtu.be)