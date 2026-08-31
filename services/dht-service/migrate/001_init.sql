CREATE TABLE IF NOT EXISTS dht_nodes
(
    node_id   BLOB PRIMARY KEY, -- 节点id
    k_bucket  INTEGER NULL,
    ip        TEXT    NULL,
    port      INTEGER NULL,
    token     BLOB,
    last_seen DATETIME
);


CREATE TABLE IF NOT EXISTS dht_infohash
(
    peer_id      BLOB PRIMARY KEY, -- 节点id
    info_hash    BLOB,
    port         int int,
    implied_port int,
    last_seen    DATETIME
);
