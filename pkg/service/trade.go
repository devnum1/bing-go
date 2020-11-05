package service

import (
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/c9s/bbgo/pkg/types"
)

type TradeService struct {
	DB *sqlx.DB
}

func NewTradeService(db *sqlx.DB) *TradeService {
	return &TradeService{db}
}

// QueryLast queries the last trade from the database
func (s *TradeService) QueryLast(ex types.ExchangeName, symbol string) (*types.Trade, error) {
	log.Infof("querying last trade exchange = %s AND symbol = %s", ex, symbol)

	rows, err := s.DB.NamedQuery(`SELECT * FROM trades WHERE exchange = :exchange AND symbol = :symbol ORDER BY gid DESC LIMIT 1`, map[string]interface{}{
		"symbol":   symbol,
		"exchange": ex,
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

func (s *TradeService) QueryForTradingFeeCurrency(ex types.ExchangeName, symbol string, feeCurrency string) ([]types.Trade, error) {
	rows, err := s.DB.NamedQuery(`SELECT * FROM trades WHERE exchange = :exchange AND (symbol = :symbol OR fee_currency = :fee_currency) ORDER BY traded_at ASC`, map[string]interface{}{
		"exchange":     ex,
		"symbol":       symbol,
		"fee_currency": feeCurrency,
	})
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	return s.scanRows(rows)
}

func (s *TradeService) Query(ex types.ExchangeName, symbol string) ([]types.Trade, error) {
	rows, err := s.DB.NamedQuery(`SELECT * FROM trades WHERE exchange = :exchange AND symbol = :symbol ORDER BY gid ASC`, map[string]interface{}{
		"exchange": ex,
		"symbol":   symbol,
	})
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	return s.scanRows(rows)
}

func (s *TradeService) scanRows(rows *sqlx.Rows) (trades []types.Trade, err error) {
	for rows.Next() {
		var trade types.Trade
		if err := rows.StructScan(&trade); err != nil {
			return trades, err
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
