package wallet

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidAmount = errors.New("invalid amount")

type Wallet struct {
	Balance   int64 `json:"balance"`
	Locked    int64 `json:"locked"`
	Available int64 `json:"available"`
}

type Holding struct {
	ItemID    string `json:"itemId"`
	ItemName  string `json:"itemName"`
	Category  string `json:"category"`
	FaceValue int64  `json:"faceValue"`
	Quantity  int64  `json:"quantity"`
	Locked    int64  `json:"locked"`
	Available int64  `json:"available"`
}

type LedgerEntry struct {
	ID          int64     `json:"id"`
	TxnID       string    `json:"txnId"`
	Account     string    `json:"account"`
	Debit       int64     `json:"debit"`
	Credit      int64     `json:"credit"`
	TradeID     *string   `json:"tradeId"` // null when not trade-related
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// GetWallet returns sql.ErrNoRows if the user has no wallet.
func (s *Store) GetWallet(ctx context.Context, userID string) (Wallet, error) {
	var w Wallet
	err := s.db.QueryRowContext(ctx,
		`SELECT balance, locked FROM wallets WHERE user_id = $1`, userID,
	).Scan(&w.Balance, &w.Locked)
	if err != nil {
		return Wallet{}, err
	}
	w.Available = w.Balance - w.Locked
	return w, nil
}

func (s *Store) GetHoldings(ctx context.Context, userID string) ([]Holding, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT h.item_id, i.name, i.category, i.face_value, h.quantity, h.locked
		FROM holdings h
		JOIN items i ON i.id = h.item_id
		WHERE h.user_id = $1 AND h.quantity > 0
		ORDER BY i.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Holding, 0)
	for rows.Next() {
		var h Holding
		if err := rows.Scan(&h.ItemID, &h.ItemName, &h.Category,
			&h.FaceValue, &h.Quantity, &h.Locked); err != nil {
			return nil, err
		}
		h.Available = h.Quantity - h.Locked
		list = append(list, h)
	}
	return list, rows.Err()
}

// GetLedger returns the user's entries, newest first.
func (s *Store) GetLedger(ctx context.Context, userID string, limit int) ([]LedgerEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, txn_id, account, debit, credit, trade_id, description, created_at
		FROM ledger_entries
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]LedgerEntry, 0)
	for rows.Next() {
		var e LedgerEntry
		if err := rows.Scan(&e.ID, &e.TxnID, &e.Account, &e.Debit, &e.Credit,
			&e.TradeID, &e.Description, &e.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, e)
	}
	return list, rows.Err()
}

// Deposit adds demo money. One transaction: wallet update + two ledger rows.
func (s *Store) Deposit(ctx context.Context, userID string, amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() // no-op after a successful Commit

	res, err := tx.ExecContext(ctx,
		`UPDATE wallets SET balance = balance + $1 WHERE user_id = $2`,
		amount, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}

	txnID := uuid.NewString()
	// system side: debit
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO ledger_entries (txn_id, account, user_id, debit, description)
		VALUES ($1, 'system_deposit', NULL, $2, 'Demo deposit')`,
		txnID, amount); err != nil {
		return err
	}
	// user side: credit
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO ledger_entries (txn_id, account, user_id, credit, description)
		VALUES ($1, 'wallet', $2, $3, 'Demo deposit')`,
		txnID, userID, amount); err != nil {
		return err
	}

	return tx.Commit()
}

// AddHolding gives a user passes/merch (used by the seed script).
func (s *Store) AddHolding(ctx context.Context, userID, itemID string, qty int64) error {
	if qty <= 0 {
		return ErrInvalidAmount
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO holdings (user_id, item_id, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, item_id)
		DO UPDATE SET quantity = holdings.quantity + EXCLUDED.quantity`,
		userID, itemID, qty)
	return err
}