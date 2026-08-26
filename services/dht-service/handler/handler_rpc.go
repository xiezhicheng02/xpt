// Package handler 实现 dht-service 对外暴露的 gRPC 接口。
package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

// GetKnownInfoHash 返回 DHT 网络已发现的所有 infohash。
// TODO(学习): 数据量大后改为分页/增量返回。
func (h *DhtHandler) GetKnownInfoHash(ctx context.Context, _ *dht.GetKnownInfoHashRequest) (*dht.GetKnownInfoHashResponse, error) {
	hashes, err := h.hashRepo.ListAll()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list infohash: %v", err)
	}
	resp := &dht.GetKnownInfoHashResponse{}
	for _, hh := range hashes {
		resp.InfoHash = append(resp.InfoHash, hh.InfoHash)
	}
	return resp, nil
}
