WIP - nothing here, move along

### what

This is the start of a multi channel funding plugin for c-lightning

### why

New release allows opening of multiple channels with a single transaction.
This is an attempt to make it less painful

### objective

The plan is to create this plugin to work either with the internal wallet,
the bitcoin core wallet or with an external wallet.

The internal wallet if used will use the bitcoin core node that c-lightning internal wallet to to provide needed utxos and change address for creating the transaction, use the c-lightning `fundchannel_start` and `fundchannel_complete` commands to get an address for each channel to provide to the transaction.

If the bitcoin core options is selected, the bitcoin core node will provide utxos and change address and continue as above

For external wallet, rpc methods will be added for providing utxo and change address info and one for sending when completed and signed by external wallet
