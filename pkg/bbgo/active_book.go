package bbgo

import (
	"github.com/sirupsen/logrus"

	"github.com/c9s/bbgo/pkg/types"
)

// LocalActiveOrderBook manages the local active order books.
type LocalActiveOrderBook struct {
	Bids *types.SyncOrderMap
	Asks *types.SyncOrderMap
}

func NewLocalActiveOrderBook() *LocalActiveOrderBook {
	return &LocalActiveOrderBook{
		Bids: types.NewSyncOrderMap(),
		Asks: types.NewSyncOrderMap(),
	}
}

func (b *LocalActiveOrderBook) Print() {
	for _, o := range b.Bids.Orders() {
		logrus.Infof("bid order: %d -> %s", o.OrderID, o.Status)
	}

	for _, o := range b.Asks.Orders() {
		logrus.Infof("ask order: %d -> %s", o.OrderID, o.Status)
	}
}

func (b *LocalActiveOrderBook) Add(orders ...types.Order) {
	for _, order := range orders {
		switch order.Side {
		case types.SideTypeBuy:
			b.Bids.Add(order)

		case types.SideTypeSell:
			b.Asks.Add(order)

		}
	}
}

func (b *LocalActiveOrderBook) NumOfBids() int {
	return b.Bids.Len()
}

func (b *LocalActiveOrderBook) NumOfAsks() int {
	return b.Asks.Len()
}

func (b *LocalActiveOrderBook) Delete(order types.Order) {
	switch order.Side {
	case types.SideTypeBuy:
		b.Bids.Delete(order.OrderID)

	case types.SideTypeSell:
		b.Asks.Delete(order.OrderID)

	}
}

// WriteOff writes off the filled order on the opposite side.
// This method does not write off order by order amount or order quantity.
func (b *LocalActiveOrderBook) WriteOff(order types.Order) bool {
	if order.Status != types.OrderStatusFilled {
		return false
	}

	switch order.Side {
	case types.SideTypeSell:
		// find the filled bid to remove
		if filledOrder, ok := b.Bids.AnyFilled(); ok {
			b.Bids.Delete(filledOrder.OrderID)
			b.Asks.Delete(order.OrderID)
			return true
		}

	case types.SideTypeBuy:
		// find the filled ask order to remove
		if filledOrder, ok := b.Asks.AnyFilled(); ok {
			b.Asks.Delete(filledOrder.OrderID)
			b.Bids.Delete(order.OrderID)
			return true
		}
	}

	return false
}

func (b *LocalActiveOrderBook) Orders() types.OrderSlice {
	return append(b.Asks.Orders(), b.Bids.Orders()...)
}
