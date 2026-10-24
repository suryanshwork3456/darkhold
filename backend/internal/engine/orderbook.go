package engine

import (
	"sort"
	"darkhold/internal/types"
)

type OrderBook struct {
	ItemID    string
	BidsMap   map[int64]*PriceLevel
	BidPrices []int64
	AskMap    map[int64]*PriceLevel
	AskPrices []int64
	OrdersMap map[string]*types.Order
}

func NewOrderBook(itemID string) *OrderBook {
	return &OrderBook{
		ItemID:    itemID,
		BidsMap:   make(map[int64]*PriceLevel),
		BidPrices: []int64{},
		AskMap:    make(map[int64]*PriceLevel),
		AskPrices: []int64{},
		OrdersMap: make(map[string]*types.Order),
	}
}

func (ob *OrderBook) AddOrder(order *types.Order) {
	ob.OrdersMap[order.ID] = order

	if order.Side == types.SideBuy {
		ob.addBid(order)
	} else {
		ob.addAsk(order)
	}
}

func (ob *OrderBook) addBid(order *types.Order) {
	price := order.Price
	pl, exists := ob.BidsMap[price]
	if !exists {
		pl = NewPriceLevel(price)
		ob.BidsMap[price] = pl
		ob.BidPrices = append(ob.BidPrices, price)
		sort.Slice(ob.BidPrices, func(i, j int) bool {
			return ob.BidPrices[i] < ob.BidPrices[j]
		})
	}
	pl.AddOrder(order)
}

func (ob *OrderBook) addAsk(order *types.Order) {
	price := order.Price
	pl, exists := ob.AskMap[price]
	if !exists {
		pl = NewPriceLevel(price)
		ob.AskMap[price] = pl
		ob.AskPrices = append(ob.AskPrices, price)
		sort.Slice(ob.AskPrices, func(i, j int) bool {
			return ob.AskPrices[i] < ob.AskPrices[j]
		})
	}
	pl.AddOrder(order)
}

func (ob *OrderBook) GetBestBid() (int64, bool) {
	if len(ob.BidPrices) == 0 {
		return 0, false
	}
	return ob.BidPrices[len(ob.BidPrices)-1], true
}

func (ob *OrderBook) GetBestAsk() (int64, bool) {
	if len(ob.AskPrices) == 0 {
		return 0, false
	}
	return ob.AskPrices[0], true
}

// CancelOrder removes an order from the book by its ID
func (ob *OrderBook) CancelOrder(orderID string) *types.Order {
	order, exists := ob.OrdersMap[orderID]
	if !exists {
		return nil
	}
	delete(ob.OrdersMap, orderID)

	if order.Side == types.SideBuy {
		if pl, ok := ob.BidsMap[order.Price]; ok {
			pl.RemoveOrder(order.ID)
			if pl.IsEmpty() {
				delete(ob.BidsMap, order.Price)
				// Remove price from BidPrices slice
				ob.BidPrices = removePrice(ob.BidPrices, order.Price)
			}
		}
	} else {
		if pl, ok := ob.AskMap[order.Price]; ok {
			pl.RemoveOrder(order.ID)
			if pl.IsEmpty() {
				delete(ob.AskMap, order.Price)
				// Remove price from AskPrices slice
				ob.AskPrices = removePrice(ob.AskPrices, order.Price)
			}
		}
	}

	return order
}


func removePrice(prices []int64, target int64) []int64 {
	for i, p := range prices {
		if p == target {
			return append(prices[:i], prices[i+1:]...)
		}
	}
	return prices
}

type PriceLevelSummary struct {
	Price    int64
	Quantity int64
}

func (ob *OrderBook) Snapshot() (bids []PriceLevelSummary, asks []PriceLevelSummary) {
	for _, price := range ob.BidPrices {
		if pl, ok := ob.BidsMap[price]; ok {
			bids = append(bids, PriceLevelSummary{Price: price, Quantity: pl.TotalQuantity()})
		}
	}
	for _, price := range ob.AskPrices {
		if pl, ok := ob.AskMap[price]; ok {
			asks = append(asks, PriceLevelSummary{Price: price, Quantity: pl.TotalQuantity()})
		}
	}
	return bids, asks
}