CREATE TABLE IF NOT EXISTS peers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    torrent_id INTEGER,
    peer_id TEXT,
    ip TEXT,
    port INTEGER,
    uploaded INTEGER DEFAULT 0,
    downloaded INTEGER DEFAULT 0,
    "left" INTEGER DEFAULT 0,
    is_seeder BOOLEAN DEFAULT 0,
    last_announce DATETIME
);
CREATE INDEX idx_peers_torrent ON peers(torrent_id);
