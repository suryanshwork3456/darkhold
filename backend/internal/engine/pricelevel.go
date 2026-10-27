package engine

import "darkhold/internal/types"

type PriceLevel struct {
	Price  int64
	Orders []*types.Order
}

func NewPriceLevel(price int64) *PriceLevel {
	return &PriceLevel{Price: price, Orders: []*types.Order{}}
}

func (pl *PriceLevel) AddOrder(order *types.Order) {
	pl.Orders = append(pl.Orders, order)
}

func (pl *PriceLevel) FirstOrder() *types.Order {
	if len(pl.Orders) == 0 {
		return nil
	}
	return pl.Orders[0]
}

func (pl *PriceLevel) RemoveFirstOrder() *types.Order {
	if len(pl.Orders) == 0 {
		return nil
	}
	first := pl.Orders[0]
	pl.Orders[0] = nil
	pl.Orders = pl.Orders[1:]
	return first
}

func (pl *PriceLevel) TotalQuantity() int64 {
	var total int64
	for _, o := range pl.Orders {
		total += o.Remaining
	}
	return total
}

func (pl *PriceLevel) IsEmpty() bool {
	return len(pl.Orders) == 0
}


func (pl *PriceLevel) RemoveOrder(orderID string) bool {
	for i, o := range pl.Orders {
		if o.ID == orderID {
			copy(pl.Orders[i:], pl.Orders[i+1:])
			pl.Orders[len(pl.Orders)-1] = nil 
			pl.Orders = pl.Orders[:len(pl.Orders)-1]
			return true
		}
	}
	return false
}