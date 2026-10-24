package engine

import (
	"sort"
	"darkhold/internal/types"
)

type OrderBook struct {
	ItemID    string
	BidsMap   map[int64]*PriceLevel
	BidPrices []int64 // sorted ascending; best bid is the LAST element
	AskMap    map[int64]*PriceLevel
	AskPrices []int64 // sorted ascending; best ask is the FIRST element
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

// AddOrder rests an order on the book. Returns false if the ID already exists.
func (ob *OrderBook) AddOrder(order *types.Order) bool {
	if _, dup := ob.OrdersMap[order.ID]; dup {
		return false
	}
	ob.OrdersMap[order.ID] = order

	if order.Side == types.Buy {
		ob.BidPrices = addToSide(ob.BidsMap, ob.BidPrices, order)
	} else {
		ob.AskPrices = addToSide(ob.AskMap, ob.AskPrices, order)
	}
	return true
}


func addToSide(levels map[int64]*PriceLevel, prices []int64, order *types.Order) []int64 {
	pl, exists := levels[order.Price]
	if !exists {
		pl = NewPriceLevel(order.Price)
		levels[order.Price] = pl
		prices = insertPrice(prices, order.Price)
	}
	pl.AddOrder(order)
	return prices
}

func insertPrice(prices []int64, price int64) []int64 {
	i := sort.Search(len(prices), func(i int) bool { return prices[i] >= price })
	prices = append(prices, 0)
	copy(prices[i+1:], prices[i:])
	prices[i] = price
	return prices
}

func removePrice(prices []int64, target int64) []int64 {
	i := sort.Search(len(prices), func(i int) bool { return prices[i] >= target })
	if i < len(prices) && prices[i] == target {
		return append(prices[:i], prices[i+1:]...)
	}
	return prices
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

// CancelOrder removes an order from the book by its ID.
func (ob *OrderBook) CancelOrder(orderID string) *types.Order {
	order, exists := ob.OrdersMap[orderID]
	if !exists {
		return nil
	}
	delete(ob.OrdersMap, orderID)

	if order.Side == types.Buy {
		if pl, ok := ob.BidsMap[order.Price]; ok {
			pl.RemoveOrder(order.ID)
			if pl.IsEmpty() {
				delete(ob.BidsMap, order.Price)
				ob.BidPrices = removePrice(ob.BidPrices, order.Price)
			}
		}
	} else {
		if pl, ok := ob.AskMap[order.Price]; ok {
			pl.RemoveOrder(order.ID)
			if pl.IsEmpty() {
				delete(ob.AskMap, order.Price)
				ob.AskPrices = removePrice(ob.AskPrices, order.Price)
			}
		}
	}
	return order
}

func (ob *OrderBook) Snapshot() types.BookSnapshot {
	snap := types.BookSnapshot{
		ItemID: ob.ItemID,
		Bids:   []types.PriceLevel{},
		Asks:   []types.PriceLevel{},
	}
	for i := len(ob.BidPrices) - 1; i >= 0; i-- { 
		price := ob.BidPrices[i]
		snap.Bids = append(snap.Bids, types.PriceLevel{
			Price:    price,
			Quantity: ob.BidsMap[price].TotalQuantity(),
		})
	}
	for _, price := range ob.AskPrices { 
		snap.Asks = append(snap.Asks, types.PriceLevel{
			Price:    price,
			Quantity: ob.AskMap[price].TotalQuantity(),
		})
	}
	return snap
}