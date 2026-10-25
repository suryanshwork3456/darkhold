package items

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// Anti-scalping cap: 120% of face value.
const collarPercent = 120

type Item struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	FaceValue   int64  `json:"faceValue"`
	MaxPrice    int64  `json:"maxPrice"`  // face value * 120%
	LastPrice   int64  `json:"lastPrice"` // last trade, 0 if none
	BestBid     int64  `json:"bestBid"`   // filled from engine later
	BestAsk     int64  `json:"bestAsk"`   // filled from engine later
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// MaxPriceFor returns the highest allowed order price for a face value.
// The order API will use this for the collar check.
func MaxPriceFor(faceValue int64) int64 {
	return faceValue * collarPercent / 100
}

const selectItem = `
	SELECT i.id, i.name, i.category, i.description, i.face_value,
	       COALESCE((SELECT t.price FROM trades t
	                 WHERE t.item_id = i.id
	                 ORDER BY t.at DESC LIMIT 1), 0)
	FROM items i`

func scanItem(row interface{ Scan(...any) error }) (Item, error) {
	var it Item
	err := row.Scan(&it.ID, &it.Name, &it.Category, &it.Description,
		&it.FaceValue, &it.LastPrice)
	if err != nil {
		return Item{}, err
	}
	it.MaxPrice = MaxPriceFor(it.FaceValue)
	return it, nil
}

// ListItems returns all items. category may be "" (all), "pass" or "merch".
func (s *Store) ListItems(ctx context.Context, category string) ([]Item, error) {
	query := selectItem
	args := []any{}
	if category != "" {
		query += ` WHERE i.category = $1`
		args = append(args, category)
	}
	query += ` ORDER BY i.created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]Item, 0) // empty slice, so JSON gives [] not null
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, it)
	}
	return list, rows.Err()
}

// GetItem returns sql.ErrNoRows if the item does not exist.
func (s *Store) GetItem(ctx context.Context, id string) (Item, error) {
	return scanItem(s.db.QueryRowContext(ctx, selectItem+` WHERE i.id = $1`, id))
}

// CreateItem is used by the seed script. Generates an ID if empty.
func (s *Store) CreateItem(ctx context.Context, it Item) (string, error) {
	if it.ID == "" {
		it.ID = uuid.NewString()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO items (id, name, category, description, face_value)
		VALUES ($1, $2, $3, $4, $5)`,
		it.ID, it.Name, it.Category, it.Description, it.FaceValue)
	return it.ID, err
}