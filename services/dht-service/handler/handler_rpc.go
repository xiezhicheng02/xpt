// Package handler 实现 dht-service 对外暴露的 gRPC 接口。
package handler

import (
	"github.com/xiezc/xpt/api/gen/dht"
	"github.com/xiezc/xpt/services/dht-service/repository"
)

// DhtHandler 实现 dht.DhtServiceServer。
// 供 web 服务调用，查询 DHT 网络已发现的 infohash。
type DhtHandler struct {
	dht.UnimplementedDhtServiceServer
	hashRepo *repository.InfoHashRepo
}

func NewDhtHandler(hashRepo *repository.InfoHashRepo) *DhtHandler {
	return &DhtHandler{hashRepo: hashRepo}
}
