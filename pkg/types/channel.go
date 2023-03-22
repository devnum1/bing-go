package types

type Channel string

const (
	BookChannel        = Channel("book")
	KLineChannel       = Channel("kline")
	BookTickerChannel  = Channel("bookTicker")
	MarketTradeChannel = Channel("trade")
	AggTradeChannel    = Channel("aggTrade")

	// channels for futures
	MarkPriceChannel        = Channel("markPrice")
	LiquidationOrderChannel = Channel("liquidationOrder")
	ContractChannel         = Channel("contract")
)
