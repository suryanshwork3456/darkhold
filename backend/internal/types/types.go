package types

import "time"

type Side string

const (
	Buy  Side = "BUY"
	Sell Side = "SELL"
)

type OrderType string

const (
	Limit  OrderType = "LIMIT"
	Market OrderType = "MARKET"
)

type Order struct {
	ID        string
	UserID    string
	ItemID    string
	Side      Side
	Type      OrderType
	Price     int64
	Quantity  int64
	Remaining int64
	CreatedAt time.Time
}

type Trade struct {
	ID          string
	ItemID      string
	BuyOrderID  string
	SellOrderID string
	BuyerID     string
	SellerID    string
	Price       int64
	Quantity    int64
	At          time.Time
}

type PriceLevel struct {
	Price    int64
	Quantity int64
}

type BookSnapshot struct {
	ItemID string
	Bids   []PriceLevel
	Asks   []PriceLevel
}

type Event struct {
	Type     string
	ItemID   string
	Snapshot *BookSnapshot
	Trade    *Trade
}
