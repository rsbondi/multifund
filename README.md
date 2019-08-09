### Overview

A multi channel funding, multi target withdraw plugin for c-lightning.

0.7.1 release of clightning allows opening of multiple channels with a single transaction.
This plugin adds RPC commands to take advantage of this.  Also, withdrawing funds to multiple
locations is a very similar process, except you know the destinations beforehand and you skip
the channel part, so it seemed logical to add as well.

### Usage

Currently there are 2 new RPC commands for channel funding

`fund_multi` and `connect_fund_multi`

`fund_multi [{"id": "02fc...", "satoshi": 20000, "announce", true}, {...}, ...]`

`connect_fund_multi` adds `"host"` and `"port"` parameters to the above

Also one command has been added for multi destination withdraw

`withdraw_multi [{"destination": ADDRESS, "satoshi": n}...]`

provide an array of objects with `destination` and `satoshi` values

### Options

There is one option that can be passed to the lightningd command line. `multi-wallet`.  `--multi-wallet=bitcoin` will use the wallet from the bitcoin core node.  Omitting this option will uset the internal c-lightning wallet, or you can be explicit with `--multi-wallet=internal`

The bitcoin core node is used for broadcasting transactions so it must be accessible even if you use clightning internal wallet.

TODO:
* Allow to set `feerate` and `minconf` on `withdraw_multi` to be consistent with `withdraw`
* Update to use rpc commands that were not available but have since been added to glightning
* Support for bitcoin cookie auth?

[demo video](https://www.youtube.com/watch?v=exDYLpTncng&feature=youtu.be)