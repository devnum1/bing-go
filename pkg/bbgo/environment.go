package bbgo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/c9s/bbgo/pkg/accounting/pnl"
	"github.com/c9s/bbgo/pkg/service"
	"github.com/c9s/bbgo/pkg/types"
	"github.com/c9s/bbgo/pkg/util"
)

var LoadedExchangeStrategies = make(map[string]SingleExchangeStrategy)
var LoadedCrossExchangeStrategies = make(map[string]CrossExchangeStrategy)

func RegisterStrategy(key string, s interface{}) {
	switch d := s.(type) {
	case SingleExchangeStrategy:
		LoadedExchangeStrategies[key] = d

	case CrossExchangeStrategy:
		LoadedCrossExchangeStrategies[key] = d
	}
}

// Environment presents the real exchange data layer
type Environment struct {
	// Notifiability here for environment is for the streaming data notification
	// note that, for back tests, we don't need notification.
	Notifiability

	TradeService *service.TradeService
	TradeSync    *service.TradeSync

	tradeScanTime time.Time
	sessions      map[string]*ExchangeSession
}

func NewEnvironment() *Environment {
	return &Environment{
		// default trade scan time
		tradeScanTime: time.Now().AddDate(0, 0, -7), // sync from 7 days ago
		sessions:      make(map[string]*ExchangeSession),
	}
}

func (environ *Environment) SyncTrades(db *sqlx.DB) *Environment {
	environ.TradeService = &service.TradeService{DB: db}
	environ.TradeSync = &service.TradeSync{
		Service: environ.TradeService,
	}

	return environ
}

func (environ *Environment) AddExchange(name string, exchange types.Exchange) (session *ExchangeSession) {
	session = NewExchangeSession(name, exchange)
	environ.sessions[name] = session
	return session
}

// Init prepares the data that will be used by the strategies
func (environ *Environment) Init(ctx context.Context) (err error) {
	for n := range environ.sessions {
		var session = environ.sessions[n]
		var markets types.MarketMap

		err = WithCache(fmt.Sprintf("%s-markets", session.Exchange.Name()), &markets, func() (interface{}, error) {
			return session.Exchange.QueryMarkets(ctx)
		})
		if err != nil {
			return err
		}

		if len(markets) == 0 {
			return errors.Errorf("market config should not be empty")
		}

		session.markets = markets

		// trade sync and market data store depends on subscribed symbols so we have to do this here.
		for symbol := range session.loadedSymbols {
			var trades []types.Trade

			if environ.TradeSync != nil {
				log.Infof("syncing trades from %s for symbol %s...", session.Exchange.Name(), symbol)
				if err := environ.TradeSync.Sync(ctx, session.Exchange, symbol, environ.tradeScanTime); err != nil {
					return err
				}

				tradingFeeCurrency := session.Exchange.PlatformFeeCurrency()
				if strings.HasPrefix(symbol, tradingFeeCurrency) {
					trades, err = environ.TradeService.QueryForTradingFeeCurrency(symbol, tradingFeeCurrency)
				} else {
					trades, err = environ.TradeService.Query(symbol)
				}

				if err != nil {
					return err
				}

				log.Infof("symbol %s: %d trades loaded", symbol, len(trades))
			}

			session.Trades[symbol] = trades

			averagePrice, err := session.Exchange.QueryAveragePrice(ctx, symbol)
			if err != nil {
				return err
			}

			session.lastPrices[symbol] = averagePrice

			marketDataStore := NewMarketDataStore(symbol)
			marketDataStore.BindStream(session.Stream)
			session.marketDataStores[symbol] = marketDataStore

			standardIndicatorSet := NewStandardIndicatorSet(symbol, marketDataStore)
			session.standardIndicatorSets[symbol] = standardIndicatorSet
		}

		now := time.Now()
		for symbol := range session.loadedSymbols {
			marketDataStore, ok := session.marketDataStores[symbol]
			if !ok {
				return errors.Errorf("symbol %s is not defined", symbol)
			}

			for interval := range types.SupportedIntervals {
				kLines, err := session.Exchange.QueryKLines(ctx, symbol, interval.String(), types.KLineQueryOptions{
					EndTime: &now,
					Limit:   500, // indicators need at least 100
				})
				if err != nil {
					return err
				}

				for _, k := range kLines {
					// let market data store trigger the update, so that the indicator could be updated too.
					marketDataStore.AddKLine(k)
				}
			}
		}

		log.Infof("querying balances...")
		balances, err := session.Exchange.QueryAccountBalances(ctx)
		if err != nil {
			return err
		}

		session.Account.UpdateBalances(balances)
		session.Account.BindStream(session.Stream)

		// update last prices
		session.Stream.OnKLineClosed(func(kline types.KLine) {
			log.Infof("kline closed: %+v", kline)
			session.lastPrices[kline.Symbol] = kline.Close
			session.marketDataStores[kline.Symbol].AddKLine(kline)
		})

		if environ.TradeService != nil {
			session.Stream.OnTradeUpdate(func(trade types.Trade) {
				if err := environ.TradeService.Insert(trade); err != nil {
					log.WithError(err).Errorf("trade insert error: %+v", trade)
				}
			})
		}

		session.Stream.OnTradeUpdate(func(trade types.Trade) {
			// append trades
			session.Trades[trade.Symbol] = append(session.Trades[trade.Symbol], trade)
		})

		// move market data store dispatch to here, use one callback to dispatch the market data
		// session.Stream.OnKLineClosed(func(kline types.KLine) { })
	}

	return nil
}

// configure notification rules
// for symbol-based routes, we should register the same symbol rules for each session.
// for session-based routes, we should set the fixed callbacks for each session
func (environ *Environment) ConfigureNotification(conf *NotificationConfig) {
	// configure routing here
	if conf.SymbolChannels != nil {
		environ.SymbolChannelRouter.AddRoute(conf.SymbolChannels)
	}
	if conf.SessionChannels != nil {
		environ.SessionChannelRouter.AddRoute(conf.SessionChannels)
	}

	if conf.Routing != nil {
		// configure passive object notification routing
		switch conf.Routing.Trade {
		case "$session":
			defaultTradeUpdateHandler := func(trade types.Trade) {
				text := util.Render(TemplateTradeReport, trade)
				environ.Notify(text, &trade)
			}
			for name := range environ.sessions {
				session := environ.sessions[name]

				// if we can route session name to channel successfully...
				channel, ok := environ.SessionChannelRouter.Route(name)
				if ok {
					session.Stream.OnTradeUpdate(func(trade types.Trade) {
						text := util.Render(TemplateTradeReport, trade)
						environ.NotifyTo(channel, text, &trade)
					})
				} else {
					session.Stream.OnTradeUpdate(defaultTradeUpdateHandler)
				}
			}

		case "$symbol":
			// configure object routes for Trade
			environ.ObjectChannelRouter.Route(func(obj interface{}) (channel string, ok bool) {
				trade, matched := obj.(*types.Trade)
				if !matched {
					return
				}
				channel, ok = environ.SymbolChannelRouter.Route(trade.Symbol)
				return
			})

			// use same handler for each session
			handler := func(trade types.Trade) {
				text := util.Render(TemplateTradeReport, trade)
				channel, ok := environ.RouteObject(&trade)
				if ok {
					environ.NotifyTo(channel, text, &trade)
				} else {
					environ.Notify(text, &trade)
				}
			}
			for _, session := range environ.sessions {
				session.Stream.OnTradeUpdate(handler)
			}
		}

		switch conf.Routing.Order {

		case "$session":
			defaultOrderUpdateHandler := func(order types.Order) {
				text := util.Render(TemplateOrderReport, order)
				environ.Notify(text, &order)
			}
			for name := range environ.sessions {
				session := environ.sessions[name]

				// if we can route session name to channel successfully...
				channel, ok := environ.SessionChannelRouter.Route(name)
				if ok {
					session.Stream.OnOrderUpdate(func(order types.Order) {
						text := util.Render(TemplateOrderReport, order)
						environ.NotifyTo(channel, text, &order)
					})
				} else {
					session.Stream.OnOrderUpdate(defaultOrderUpdateHandler)
				}
			}

		case "$symbol":
			// add object route
			environ.ObjectChannelRouter.Route(func(obj interface{}) (channel string, ok bool) {
				order, matched := obj.(*types.Order)
				if !matched {
					return
				}
				channel, ok = environ.SymbolChannelRouter.Route(order.Symbol)
				return
			})

			// use same handler for each session
			handler := func(order types.Order) {
				text := util.Render(TemplateOrderReport, order)
				channel, ok := environ.RouteObject(&order)
				if ok {
					environ.NotifyTo(channel, text, &order)
				} else {
					environ.Notify(text, &order)
				}
			}
			for _, session := range environ.sessions {
				session.Stream.OnOrderUpdate(handler)
			}
		}

		switch conf.Routing.SubmitOrder {
		case "$symbol":
			// add object route
			environ.ObjectChannelRouter.Route(func(obj interface{}) (channel string, ok bool) {
				order, matched := obj.(*types.SubmitOrder)
				if !matched {
					return
				}

				channel, ok = environ.SymbolChannelRouter.Route(order.Symbol)
				return
			})

		}

		// currently not used
		switch conf.Routing.PnL {
		case "$symbol":
			environ.ObjectChannelRouter.Route(func(obj interface{}) (channel string, ok bool) {
				report, matched := obj.(*pnl.AverageCostPnlReport)
				if !matched {
					return
				}
				channel, ok = environ.SymbolChannelRouter.Route(report.Symbol)
				return
			})
		}

	}
}

// SyncTradesFrom overrides the default trade scan time (-7 days)
func (environ *Environment) SyncTradesFrom(t time.Time) *Environment {
	environ.tradeScanTime = t
	return environ
}

func (environ *Environment) Connect(ctx context.Context) error {
	for n := range environ.sessions {
		// avoid using the placeholder variable for the session because we use that in the callbacks
		var session = environ.sessions[n]
		var logger = log.WithField("session", n)

		if len(session.Subscriptions) == 0 {
			logger.Warnf("no subscriptions, exchange session %s will not be connected", session.Name)
			continue
		}

		// add the subscribe requests to the stream
		for _, s := range session.Subscriptions {
			logger.Infof("subscribing %s %s %v", s.Symbol, s.Channel, s.Options)
			session.Stream.Subscribe(s.Channel, s.Symbol, s.Options)
		}

		logger.Infof("connecting session %s...", session.Name)
		if err := session.Stream.Connect(ctx); err != nil {
			return err
		}
	}

	return nil
}

func BatchQueryKLineWindows(ctx context.Context, e types.Exchange, symbol string, intervals []string, startTime, endTime time.Time) (map[string]types.KLineWindow, error) {
	batch := &types.ExchangeBatchProcessor{Exchange: e}
	klineWindows := map[string]types.KLineWindow{}
	for _, interval := range intervals {
		kLines, err := batch.BatchQueryKLines(ctx, symbol, interval, startTime, endTime)
		if err != nil {
			return klineWindows, err
		}

		klineWindows[interval] = kLines
	}

	return klineWindows, nil
}
