package model

type Peer struct {
	ID           int64
	TorrentID    int64
	PeerID       string
	IP           string
	Port         int
	Uploaded     int64
	Downloaded   int64
	Left         int64
	IsSeeder     bool
	LastAnnounce string
}
