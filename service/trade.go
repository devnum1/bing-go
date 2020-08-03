package service

import (
	"context"
	"github.com/c9s/bbgo/pkg/bbgo/exchange/binance"
	"github.com/c9s/bbgo/pkg/bbgo/types"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"time"
)

type TradeSync struct {
	Service  *TradeService
	Exchange *binance.Exchange
}

func (s *TradeSync) Sync(ctx context.Context, symbol string, startTime time.Time) error {
	lastTrade, err := s.Service.QueryLast(symbol)
	if err != nil {
		return err
	}

	var lastID int64 = 0
	if lastTrade != nil {
		lastID = lastTrade.ID
		startTime = lastTrade.Time

		log.Infof("found last trade, start from lastID = %d since %s", lastTrade.ID, startTime)
	}

	trades, err := s.Exchange.BatchQueryTrades(ctx, symbol, &binance.TradeQueryOptions{
		StartTime:   &startTime,
		Limit:       1000,
		LastTradeID: lastID,
	})
	if err != nil {
		return err
	}

	for _, trade := range trades {
		if err := s.Service.Insert(trade); err != nil {
			return err
		}
	}

	return nil
}

type TradeService struct {
	DB *sqlx.DB
}

func NewTradeService(db *sqlx.DB) *TradeService {
	return &TradeService{db}
}

// QueryLast queries the last trade from the database
func (s *TradeService) QueryLast(symbol string) (*types.Trade, error) {
	log.Infof("querying last trade symbol = %s", symbol)

	rows, err := s.DB.NamedQuery(`SELECT * FROM trades WHERE symbol = :symbol ORDER BY gid DESC LIMIT 1`, map[string]interface{}{
		"symbol": symbol,
	})
	if err != nil {
		return nil, errors.Wrap(err, "query last trade error")
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	defer rows.Close()

	if rows.Next() {
		var trade types.Trade
		err = rows.StructScan(&trade)
		return &trade, err
	}

	return nil, rows.Err()
}

func (s *TradeService) Query(symbol string) (trades []types.Trade, err error) {
	rows, err := s.DB.NamedQuery(`SELECT * FROM trades WHERE symbol = :symbol ORDER BY gid ASC`, map[string]interface{}{
		"symbol": symbol,
	})
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var trade types.Trade
		if err := rows.StructScan(&trade); err != nil {
			return nil, err
		}

		trades = append(trades, trade)
	}

	return trades, rows.Err()
}

func (s *TradeService) Insert(trade types.Trade) error {
	_, err := s.DB.NamedExec(`
			INSERT INTO trades (id, exchange, symbol, price, quantity, quote_quantity, side, is_buyer, is_maker, fee, fee_currency, traded_at)
			VALUES (:id, :exchange, :symbol, :price, :quantity, :quote_quantity, :side, :is_buyer, :is_maker, :fee, :fee_currency, :traded_at)`,
		trade)
	return err
}
