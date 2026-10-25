CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    email         TEXT UNIQUE NOT NULL,
    name          TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS items (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    category    TEXT NOT NULL CHECK (category IN ('pass', 'merch')),
    description TEXT NOT NULL DEFAULT '',
    face_value  BIGINT NOT NULL CHECK (face_value > 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS wallets (
    user_id TEXT PRIMARY KEY REFERENCES users(id),
    balance BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0),
    locked  BIGINT NOT NULL DEFAULT 0 CHECK (locked >= 0),
    CHECK (locked <= balance)
);

CREATE TABLE IF NOT EXISTS holdings (
    user_id  TEXT NOT NULL REFERENCES users(id),
    item_id  TEXT NOT NULL REFERENCES items(id),
    quantity BIGINT NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    locked   BIGINT NOT NULL DEFAULT 0 CHECK (locked >= 0),
    PRIMARY KEY (user_id, item_id),
    CHECK (locked <= quantity)
);

CREATE TABLE IF NOT EXISTS orders (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id),
    item_id    TEXT NOT NULL REFERENCES items(id),
    side       TEXT NOT NULL CHECK (side IN ('buy', 'sell')),
    type       TEXT NOT NULL CHECK (type IN ('limit', 'market')),
    price      BIGINT NOT NULL CHECK (price >= 0),
    quantity   BIGINT NOT NULL CHECK (quantity > 0),
    remaining  BIGINT NOT NULL CHECK (remaining >= 0 AND remaining <= quantity),
    status     TEXT NOT NULL DEFAULT 'open'
               CHECK (status IN ('open', 'filled', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS trades (
    id            TEXT PRIMARY KEY,
    item_id       TEXT NOT NULL REFERENCES items(id),
    buy_order_id  TEXT NOT NULL REFERENCES orders(id),
    sell_order_id TEXT NOT NULL REFERENCES orders(id),
    buyer_id      TEXT NOT NULL REFERENCES users(id),
    seller_id     TEXT NOT NULL REFERENCES users(id),
    price         BIGINT NOT NULL CHECK (price > 0),
    quantity      BIGINT NOT NULL CHECK (quantity > 0),
    at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (buyer_id <> seller_id)
);

-- Double-entry ledger: every money movement is 2+ rows sharing a txn_id.
-- For each txn_id: SUM(debit) = SUM(credit).
CREATE TABLE IF NOT EXISTS ledger_entries (
    id          BIGSERIAL PRIMARY KEY,
    txn_id      TEXT NOT NULL,
    account     TEXT NOT NULL,              -- 'wallet', 'system_deposit', ...
    user_id     TEXT REFERENCES users(id),  -- NULL for system accounts
    debit       BIGINT NOT NULL DEFAULT 0 CHECK (debit >= 0),
    credit      BIGINT NOT NULL DEFAULT 0 CHECK (credit >= 0),
    trade_id    TEXT REFERENCES trades(id),
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK ((debit = 0) <> (credit = 0))     -- exactly one side is non-zero
);

CREATE INDEX IF NOT EXISTS idx_orders_user    ON orders(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_item    ON orders(item_id, status);
CREATE INDEX IF NOT EXISTS idx_trades_item    ON trades(item_id, at DESC);
CREATE INDEX IF NOT EXISTS idx_ledger_user    ON ledger_entries(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ledger_txn     ON ledger_entries(txn_id);