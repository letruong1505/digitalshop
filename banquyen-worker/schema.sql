CREATE TABLE IF NOT EXISTS orders (
  code TEXT PRIMARY KEY,
  token TEXT NOT NULL,
  product TEXT NOT NULL,
  plan TEXT NOT NULL,
  amount INTEGER NOT NULL,
  machine TEXT NOT NULL,
  shop TEXT NOT NULL,
  phone TEXT NOT NULL,
  email TEXT NOT NULL DEFAULT '',
  mst TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending',
  created_at INTEGER NOT NULL,
  paid_at INTEGER,
  paid_amount INTEGER,
  txn_id TEXT,
  license_key TEXT,
  license_id TEXT,
  exp TEXT,
  ip_hash TEXT,
  note TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_orders_machine ON orders(product, machine, status);
CREATE INDEX IF NOT EXISTS idx_orders_ip ON orders(ip_hash, created_at);
CREATE TABLE IF NOT EXISTS txns (
  id TEXT PRIMARY KEY,
  at INTEGER NOT NULL,
  amount INTEGER NOT NULL,
  content TEXT NOT NULL,
  order_code TEXT,
  result TEXT NOT NULL
);
