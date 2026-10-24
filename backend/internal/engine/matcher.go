package engine

import (
	"time"

	"darkhold/internal/types"
)

// Match processes an incoming order against the order book, executing trades
// and returning a slice of generated trades.
func Match(book *OrderBook, order *types.Order) []types.Trade {
	trades := []types.Trade{}

	// Ensure remaining quantity is initialized
	if order.Remaining == 0 {
		order.Remaining = order.Quantity
	}

	for order.Remaining > 0 {
		var bestPrice int64
		var hasPrice bool
		var levels map[int64]*PriceLevel
		var priceSlice *[]int64

		// 1 & 2. Determine target side and get best opposite price level
		if order.Side == types.Buy {
			bestPrice, hasPrice = book.GetBestAsk()
			levels = book.AskMap
			priceSlice = &book.AskPrices
		} else {
			bestPrice, hasPrice = book.GetBestBid()
			levels = book.BidsMap
			priceSlice = &book.BidPrices
		}

		// 3. Stop if no opposite price level exists, or limit price doesn't cross
		if !hasPrice {
			break
		}
		if order.Type == types.Limit {
			if order.Side == types.Buy && order.Price < bestPrice {
				break
			}
			if order.Side == types.Sell && order.Price > bestPrice {
				break
			}
		}

		pl, ok := levels[bestPrice]
		if !ok || pl.IsEmpty() {
			break
		}

		restingOrder := pl.FirstOrder()
		if restingOrder == nil {
			break
		}

		// 4. Trade quantity is the smaller of the two remaining amounts
		tradeQty := order.Remaining
		if restingOrder.Remaining < tradeQty {
			tradeQty = restingOrder.Remaining
		}

		// 5. Subtract quantity from both orders
		order.Remaining -= tradeQty
		restingOrder.Remaining -= tradeQty

		// Determine buyer/seller IDs based on order sides
		var buyOrderID, sellOrderID, buyerID, sellerID string
		if order.Side == types.Buy {
			buyOrderID = order.ID
			sellOrderID = restingOrder.ID
			buyerID = order.UserID
			sellerID = restingOrder.UserID
		} else {
			buyOrderID = restingOrder.ID
			sellOrderID = order.ID
			buyerID = restingOrder.UserID
			sellerID = order.UserID
		}

		// Create Trade record (Trade price is the resting order's price)
		trades = append(trades, types.Trade{
			ID:          generateTradeID(),
			ItemID:      book.ItemID,
			BuyOrderID:  buyOrderID,
			SellOrderID: sellOrderID,
			BuyerID:     buyerID,
			SellerID:    sellerID,
			Price:       restingOrder.Price,
			Quantity:    tradeQty,
			At:          time.Now(),
		})

		// If resting order is fully filled, remove it from the book
		if restingOrder.Remaining == 0 {
			delete(book.OrdersMap, restingOrder.ID)
			pl.RemoveFirstOrder()

			// If the price level is empty, remove it from map and price slice
			if pl.IsEmpty() {
				delete(levels, bestPrice)
				*priceSlice = removePrice(*priceSlice, bestPrice)
			}
		}
	}

	// 7. If limit order has quantity left, add it to the book. Market orders drop remainder.
	if order.Type == types.Limit && order.Remaining > 0 {
		book.AddOrder(order)
	}

	return trades
}

// Simple unique trade ID generator helper
func generateTradeID() string {
	return "trade_" + time.Now().Format("20060102150405.000000")
}