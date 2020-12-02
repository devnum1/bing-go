package bbgo

import (
	"github.com/c9s/bbgo/pkg/indicator"
	"github.com/c9s/bbgo/pkg/types"
)

type StandardIndicatorSet struct {
	Symbol string
	// Standard indicators
	// interval -> window
	sma  map[types.IntervalWindow]*indicator.SMA
	ewma map[types.IntervalWindow]*indicator.EWMA
	boll map[types.IntervalWindow]*indicator.BOLL

	store *MarketDataStore
}

func NewStandardIndicatorSet(symbol string, store *MarketDataStore) *StandardIndicatorSet {
	set := &StandardIndicatorSet{
		Symbol: symbol,
		sma:    make(map[types.IntervalWindow]*indicator.SMA),
		ewma:   make(map[types.IntervalWindow]*indicator.EWMA),
		boll:   make(map[types.IntervalWindow]*indicator.BOLL),
		store:  store,
	}

	// let us pre-defined commonly used intervals
	for interval := range types.SupportedIntervals {
		for _, window := range []int{7, 25, 99} {
			iw := types.IntervalWindow{Interval: interval, Window: window}
			set.sma[iw] = &indicator.SMA{IntervalWindow: iw}
			set.sma[iw].Bind(store)

			set.ewma[iw] = &indicator.EWMA{IntervalWindow: iw}
			set.ewma[iw].Bind(store)
		}

		// setup boll indicator, we may refactor boll indicator by subscribing SMA indicator,
		// however, since general used BOLLINGER band use window 21, which is not in the existing SMA indicator sets.
		// Pull out the bandwidth configuration as the boll Key
		iw := types.IntervalWindow{Interval: interval, Window: 21}
		set.boll[iw] = &indicator.BOLL{IntervalWindow: iw, K: 2.0}
		set.boll[iw].Bind(store)
	}

	return set
}

// BOLL returns the bollinger band indicator of the given interval and the window,
// Please note that the K for std dev is fixed and defaults to 2.0
func (set *StandardIndicatorSet) BOLL(iw types.IntervalWindow, bandWidth float64) *indicator.BOLL {
	inc, ok := set.boll[iw]
	if !ok {
		inc := &indicator.BOLL{IntervalWindow: iw, K: bandWidth}
		inc.Bind(set.store)
		set.boll[iw] = inc
	}

	return inc
}

// SMA returns the simple moving average indicator of the given interval and the window size.
func (set *StandardIndicatorSet) SMA(iw types.IntervalWindow) *indicator.SMA {
	inc, ok := set.sma[iw]
	if !ok {
		inc := &indicator.SMA{IntervalWindow: iw}
		inc.Bind(set.store)
		set.sma[iw] = inc
	}

	return inc
}

// GetEWMA returns the exponential weighed moving average indicator of the given interval and the window size.
func (set *StandardIndicatorSet) EWMA(iw types.IntervalWindow) *indicator.EWMA {
	inc, ok := set.ewma[iw]
	if !ok {
		inc := &indicator.EWMA{IntervalWindow: iw}
		inc.Bind(set.store)
		set.ewma[iw] = inc
	}

	return inc
}

// ExchangeSession presents the exchange connection Session
// It also maintains and collects the data returned from the stream.
type ExchangeSession struct {
	// exchange Session based notification system
	// we make it as a value field so that we can configure it separately
	Notifiability

	// Exchange Session name
	Name string

	// The exchange account states
	Account *types.Account

	// Stream is the connection stream of the exchange
	Stream types.Stream

	Subscriptions map[types.Subscription]types.Subscription

	Exchange types.Exchange

	// markets defines market configuration of a symbol
	markets map[string]types.Market

	// startPrices is used for backtest
	startPrices map[string]float64

	lastPrices map[string]float64

	// Trades collects the executed trades from the exchange
	// map: symbol -> []trade
	Trades map[string][]types.Trade

	// marketDataStores contains the market data store of each market
	marketDataStores map[string]*MarketDataStore

	// standard indicators of each market
	standardIndicatorSets map[string]*StandardIndicatorSet

	loadedSymbols map[string]struct{}
}

func NewExchangeSession(name string, exchange types.Exchange) *ExchangeSession {
	return &ExchangeSession{
		Notifiability: Notifiability{
			SymbolChannelRouter:  NewPatternChannelRouter(nil),
			SessionChannelRouter: NewPatternChannelRouter(nil),
			ObjectChannelRouter:  NewObjectChannelRouter(),
		},

		Name:          name,
		Exchange:      exchange,
		Stream:        exchange.NewStream(),
		Subscriptions: make(map[types.Subscription]types.Subscription),
		Account:       &types.Account{},
		Trades:        make(map[string][]types.Trade),

		markets:               make(map[string]types.Market),
		startPrices:           make(map[string]float64),
		lastPrices:            make(map[string]float64),
		marketDataStores:      make(map[string]*MarketDataStore),
		standardIndicatorSets: make(map[string]*StandardIndicatorSet),

		loadedSymbols: make(map[string]struct{}),
	}
}

func (session *ExchangeSession) StandardIndicatorSet(symbol string) (*StandardIndicatorSet, bool) {
	set, ok := session.standardIndicatorSets[symbol]
	return set, ok
}

// MarketDataStore returns the market data store of a symbol
func (session *ExchangeSession) MarketDataStore(symbol string) (s *MarketDataStore, ok bool) {
	s, ok = session.marketDataStores[symbol]
	return s, ok
}

func (session *ExchangeSession) StartPrice(symbol string) (price float64, ok bool) {
	price, ok = session.startPrices[symbol]
	return price, ok
}

func (session *ExchangeSession) LastPrice(symbol string) (price float64, ok bool) {
	price, ok = session.lastPrices[symbol]
	return price, ok
}

func (session *ExchangeSession) Market(symbol string) (market types.Market, ok bool) {
	market, ok = session.markets[symbol]
	return market, ok
}

// Subscribe save the subscription info, later it will be assigned to the stream
func (session *ExchangeSession) Subscribe(channel types.Channel, symbol string, options types.SubscribeOptions) *ExchangeSession {
	sub := types.Subscription{
		Channel: channel,
		Symbol:  symbol,
		Options: options,
	}

	// add to the loaded symbol table
	session.loadedSymbols[symbol] = struct{}{}
	session.Subscriptions[sub] = sub
	return session
}
