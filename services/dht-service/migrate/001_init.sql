CREATE TABLE IF NOT EXISTS dht_nodes
(
    node_id   BLOB PRIMARY KEY, -- 节点id
    k_bucket  INTEGER  NOT NULL,
    ip        TEXT     NULL,
    port      INTEGER  NULL,
    token     BLOB,
    last_seen DATETIME NOT NULL
);

create table main.dht_infohash
(
    peer_id      BLOB not null,
    info_hash    BLOB not null,
    port         int int,
    implied_port int,
    last_seen    DATETIME,
    constraint dht_infohash_pk
        primary key (peer_id, info_hash)
);

