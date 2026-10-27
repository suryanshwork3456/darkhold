package settlement

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"darkhold/internal/holdings"
	"darkhold/internal/types"
)

var ErrInsufficientFunds = errors.New("insufficient funds")

// Migrate creates the trades table and the ledger table.
// ASSUMES a wallets table already exists: wallets(user_id, balance, locked).
// If your wallet module already has a ledger table, point the INSERTs below at it instead.
func Migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS trades (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			engine_trade_id TEXT NOT NULL,
			item_id         TEXT NOT NULL,
			buy_order_id    TEXT NOT NULL,
			sell_order_id   TEXT NOT NULL,
			buyer_id        TEXT NOT NULL,
			seller_id       TEXT NOT NULL,
			price           INTEGER NOT NULL,
			quantity        INTEGER NOT NULL,
			created_at      TEXT NOT NULL
		)`,
		// Double entry: every trade writes one row with debit and one with credit.
		// debit = money out, credit = money in. Total debits must always equal total credits.
		`CREATE TABLE IF NOT EXISTS ledger_entries (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			trade_id   INTEGER NOT NULL,
			account    TEXT NOT NULL,
			debit      INTEGER NOT NULL DEFAULT 0,
			credit     INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	return nil
}

// LockFunds reserves money for a buy order. Fails if available money (balance - locked) is too low.
func LockFunds(tx *sql.Tx, userID string, amount int64) error {
	res, err := tx.Exec(
		`UPDATE wallets SET locked = locked + ? WHERE user_id = ? AND balance - locked >= ?`,
		amount, userID, amount)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrInsufficientFunds
	}
	return nil
}

// ReleaseFunds gives back reserved money (order cancelled or not filled).
func ReleaseFunds(tx *sql.Tx, userID string, amount int64) error {
	return exec(tx, `UPDATE wallets SET locked = locked - ? WHERE user_id = ?`, amount, userID)
}

// Settle moves money and tickets for every trade in ONE database transaction.
// If anything fails, nothing is changed.
func Settle(db *sql.DB, trades []types.Trade) error {
	if len(trades) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // does nothing after a successful Commit

	for _, t := range trades {
		if err := settleOne(tx, t); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func settleOne(tx *sql.Tx, t types.Trade) error {
	total := t.Price * t.Quantity

	// The buyer locked money at their own order's lock price, which can be
	// higher than the trade price. Releasing the full lock while only
	// charging the trade price is what refunds the difference.
	var lockPrice int64
	if err := tx.QueryRow(`SELECT lock_price FROM orders WHERE id = ?`, t.BuyOrderID).Scan(&lockPrice); err != nil {
		return fmt.Errorf("buy order %s: %w", t.BuyOrderID, err)
	}

	// 1. Money: buyer pays, seller receives.
	if err := exec(tx,
		`UPDATE wallets SET balance = balance - ?, locked = locked - ? WHERE user_id = ?`,
		total, lockPrice*t.Quantity, t.BuyerID); err != nil {
		return err
	}
	if err := exec(tx, `UPDATE wallets SET balance = balance + ? WHERE user_id = ?`, total, t.SellerID); err != nil {
		return err
	}

	// 2. Tickets: seller loses (they were locked), buyer gains.
	if err := holdings.TakeLocked(tx, t.SellerID, t.ItemID, t.Quantity); err != nil {
		return err
	}
	if err := holdings.Add(tx, t.BuyerID, t.ItemID, t.Quantity); err != nil {
		return err
	}

	// 3. Progress on both orders.
	for _, orderID := range []string{t.BuyOrderID, t.SellOrderID} {
		if err := exec(tx, `
			UPDATE orders
			SET remaining = remaining - ?,
			    status = CASE WHEN remaining - ? = 0 THEN 'FILLED' ELSE 'PARTIAL' END
			WHERE id = ?`, t.Quantity, t.Quantity, orderID); err != nil {
			return err
		}
	}

	// 4. Trade record. We use our own id because the engine's counter restarts on every run.
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := tx.Exec(`
		INSERT INTO trades (engine_trade_id, item_id, buy_order_id, sell_order_id, buyer_id, seller_id, price, quantity, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ItemID, t.BuyOrderID, t.SellOrderID, t.BuyerID, t.SellerID, t.Price, t.Quantity, now)
	if err != nil {
		return err
	}
	tradeID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	// 5. Double-entry ledger: one debit and one credit of the same amount.
	if err := exec(tx,
		`INSERT INTO ledger_entries (trade_id, account, debit, credit, created_at) VALUES (?, ?, ?, 0, ?)`,
		tradeID, "user:"+t.BuyerID, total, now); err != nil {
		return err
	}
	return exec(tx,
		`INSERT INTO ledger_entries (trade_id, account, debit, credit, created_at) VALUES (?, ?, 0, ?, ?)`,
		tradeID, "user:"+t.SellerID, total, now)
}

// exec runs a statement and fails if it changed no rows.
func exec(tx *sql.Tx, query string, args ...any) error {
	res, err := tx.Exec(query, args...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("settlement: no rows changed by: %s", query)
	}
	return nil
}