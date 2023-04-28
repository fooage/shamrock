package rpc_api

import (
	"context"
	"errors"

	"github.com/fooage/shamrock/core/filestore"
	"github.com/fooage/shamrock/core/raft"
	"github.com/fooage/shamrock/proto/proto_gen/block_service"
	"go.uber.org/zap"
)

type handler struct {
	fileStorage filestore.FileStorage
	raftCluster raft.Cluster
	logger      *zap.Logger
	// implement abstract block service method
	block_service.UnimplementedBlockServiceServer
}

func generateHandler(logger *zap.Logger, fileStorage filestore.FileStorage, raftCluster raft.Cluster) *handler {
	return &handler{
		fileStorage: fileStorage,
		raftCluster: raftCluster,
		logger:      logger,
	}
}

func (h *handler) GetChunk(ctx context.Context, req *block_service.GetChunkReq) (*block_service.GetChunkResp, error) {
	if value, ok := h.fileStorage.Lookup(req.UniqueKey); ok {
		return &block_service.GetChunkResp{
			UniqueKey: req.UniqueKey, Binary: value,
		}, nil
	} else {
		h.logger.Error("request chunk not found", zap.String("hash", req.UniqueKey))
		return nil, errors.New("")
	}
}

func (h *handler) SaveChunk(ctx context.Context, req *block_service.SaveChunkReq) (*block_service.SaveChunkResp, error) {
	err := h.fileStorage.Propose(filestore.SaveCommand, req.UniqueKey, req.Binary)
	if err != nil {
		h.logger.Error("propose chunk in cluster error", zap.Error(err), zap.String("hash", req.UniqueKey))
		return nil, err
	}
	return &block_service.SaveChunkResp{}, nil
}

func (h *handler) RemoveChunk(ctx context.Context, req *block_service.RemoveChunkReq) (*block_service.RemoveChunkResp, error) {
	err := h.fileStorage.Propose(filestore.RemoveCommand, req.UniqueKey, nil)
	if err != nil {
		h.logger.Error("remove chunk in cluster error", zap.Error(err), zap.String("hash", req.UniqueKey))
		return nil, err
	}
	return &block_service.RemoveChunkResp{}, nil
}
