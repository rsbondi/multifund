WIP - Not ready for prime time

### what

This is the start of a multi channel funding plugin for c-lightning

### why

New release of clightning allows opening of multiple channels with a single transaction.
This is an attempt to make it less painful

### status

Currently there are 2 new RPC commands for channel funding

`fund_multi` and `connect_fund_multi`

`fund_multi [{"id": "02fc...", "satoshi": 20000, "announce", true}, {...}, ...]`

`connect_fund_multi` adds `"host"` and `"port"` parameters to the above

Also one command has been added for multi destination withdraw

`withdraw_multi [{"destination": ADDRESS, "satoshi": n},,,]`

provide an array of objects with `destination` and `satoshi` values

Using either bitcoin core wallet or internal clightning wallet types seem to be working

The bitcoin core node is used for broadcasting transactions so it must be accessible even if you use clightning internal wallet.

TODO:
* Allow to set `feerate` and `minconf` on `withdraw_multi` to be consistent with `withdraw`

[demo video](https://www.youtube.com/watch?v=exDYLpTncng&feature=youtu.be)