CREATE TABLE IF NOT EXISTS dht_nodes
(
    node_id   BLOB PRIMARY KEY, -- 节点id
    k_bucket  INTEGER  NOT NULL,
    ip        TEXT     NULL,
    port      INTEGER  NULL,
    token     BLOB,
    last_seen DATETIME NOT NULL
);


create table dht_infohash
(
    id           INTEGER  not null primary key autoincrement,
    peer_id      BLOB     not null,
    info_hash    BLOB     not null,
    port         INTEGER,
    implied_port INTEGER,
    last_seen    DATETIME NOT NULL,

    constraint dht_infohash_uk
        unique (peer_id, info_hash)
);


