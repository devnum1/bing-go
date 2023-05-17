package bitgetapi

//go:generate -command GetRequest requestgen -method GET -responseType .APIResponse -responseDataField Data
//go:generate -command PostRequest requestgen -method POST -responseType .APIResponse -responseDataField Data

import (
	"github.com/c9s/requestgen"

	"github.com/c9s/bbgo/pkg/fixedpoint"
)

type Symbol struct {
	Symbol              string           `json:"symbol"`
	SymbolName          string           `json:"symbolName"`
	BaseCoin            string           `json:"baseCoin"`
	QuoteCoin           string           `json:"quoteCoin"`
	MinTradeAmount      fixedpoint.Value `json:"minTradeAmount"`
	MaxTradeAmount      fixedpoint.Value `json:"maxTradeAmount"`
	TakerFeeRate        fixedpoint.Value `json:"takerFeeRate"`
	MakerFeeRate        fixedpoint.Value `json:"makerFeeRate"`
	PriceScale          fixedpoint.Value `json:"priceScale"`
	QuantityScale       fixedpoint.Value `json:"quantityScale"`
	MinTradeUSDT        fixedpoint.Value `json:"minTradeUSDT"`
	Status              string           `json:"status"`
	BuyLimitPriceRatio  fixedpoint.Value `json:"buyLimitPriceRatio"`
	SellLimitPriceRatio fixedpoint.Value `json:"sellLimitPriceRatio"`
}

//go:generate GetRequest -url "/api/spot/v1/public/products" -type GetSymbolsRequest -responseDataType []Symbol
type GetSymbolsRequest struct {
	client requestgen.AuthenticatedAPIClient
}

func (c *RestClient) NewGetSymbolsRequest() *GetSymbolsRequest {
	return &GetSymbolsRequest{client: c}
}
