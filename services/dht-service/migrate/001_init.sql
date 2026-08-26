CREATE TABLE IF NOT EXISTS dht_nodes (
	node_id BLOB PRIMARY KEY,      -- 节点id
    k_bucket INTEGER NULL,
    ip4 TEXT NULL,
	port4 INTEGER NULL,
	token4  TEXT NULL,
	ip6 TEXT NULL,
	port6 INTEGER NULL,
	token6  TEXT NULL,
	last_seen DATETIME
);


CREATE TABLE IF NOT EXISTS dht_infohash (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	node_id TEXT ,
	info_hash TEXT UNIQUE,
	discover_at DATETIME
);
