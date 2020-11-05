package service

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/c9s/bbgo/pkg/types"
)

type SyncService struct {
	TradeService *TradeService
	OrderService *OrderService
}

func (s *SyncService) SyncOrders(ctx context.Context, exchange types.Exchange, symbol string, startTime time.Time) error {
	lastOrder, err := s.OrderService.QueryLast(exchange.Name(), symbol)
	if err != nil {
		return err
	}

	var lastID uint64 = 0
	if lastOrder != nil {
		lastID = lastOrder.OrderID
		startTime = lastOrder.CreationTime

		logrus.Infof("found last order, start from lastID = %d since %s", lastID, startTime)
	}

	batch := &types.ExchangeBatchProcessor{Exchange: exchange}
	ordersC, errC := batch.BatchQueryClosedOrders(ctx, symbol, startTime, time.Now(), lastID)
	for order := range ordersC {
		select {

		case <-ctx.Done():
			return ctx.Err()

		case err := <-errC:
			return err

		default:

		}

		if err := s.OrderService.Insert(order); err != nil {
			return err
		}
	}

	return nil
}

func (s *SyncService) SyncTrades(ctx context.Context, exchange types.Exchange, symbol string, startTime time.Time) error {
	lastTrade, err := s.TradeService.QueryLast(exchange.Name(), symbol)
	if err != nil {
		return err
	}

	var lastID int64 = 0
	if lastTrade != nil {
		lastID = lastTrade.ID
		startTime = lastTrade.Time

		logrus.Infof("found last trade, start from lastID = %d since %s", lastID, startTime)
	}

	batch := &types.ExchangeBatchProcessor{Exchange: exchange}
	trades, err := batch.BatchQueryTrades(ctx, symbol, &types.TradeQueryOptions{
		StartTime:   &startTime,
		Limit:       200,
		LastTradeID: lastID,
	})
	if err != nil {
		return err
	}

	for _, trade := range trades {
		if err := s.TradeService.Insert(trade); err != nil {
			return err
		}
	}

	return nil
}
