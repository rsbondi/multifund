package main

// this is where the missing glightning rpc calls are implemented, eventually to be moved to glightning

type FundChannelStartRequest struct {
	Id       string `json:"id"`
	Amount   uint64 `json:"satoshi"`
	FeeRate  string `json:"feerate,omitempty"`
	Announce bool   `json:"announce"`
}

func (f *FundChannelStartRequest) Name() string {
	return "fundchannel_start"
}

type FundChannelStartResponse struct {
	FundingAddress string `json:"funding_address"`
}

func FundChannelStart(id string, amt uint64) (*FundChannelStartResponse, error) {
	result := &FundChannelStartResponse{}
	req := &FundChannelStartRequest{}
	req.Id = id
	req.Amount = amt
	req.Announce = true
	err := lightning.Request(req, result)
	return result, err
}

type FundChannelCompleteRequest struct {
	Id    string `json:"id"`
	Txid  string `json:"txid"`
	Txout int    `json:"txout"`
}

func (f *FundChannelCompleteRequest) Name() string {
	return "fundchannel_complete"
}

type FundChannelCompleteResponse struct {
	ChannelId string `json:"channel_id"`
	Secured   bool   `json:"commitments_secured"`
}

func FundChannelComplete(id string, txid string, vout int) (*FundChannelCompleteResponse, error) {
	result := &FundChannelCompleteResponse{}
	req := &FundChannelCompleteRequest{}
	req.Id = id
	req.Txid = txid
	req.Txout = vout
	err := lightning.Request(req, result)
	return result, err
}
