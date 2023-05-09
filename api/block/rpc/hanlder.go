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

func (h *handler) FetchBlock(ctx context.Context, req *block_service.FetchBlockReq) (*block_service.FetchBlockResp, error) {
	if value, ok := h.fileStorage.Lookup(req.UniqueKey); ok {
		return &block_service.FetchBlockResp{
			Entry: &block_service.BlockEntry{
				UniqueKey: value.UniqueKey,
				Hash:      value.Hash,
				Binary:    value.Binary,
			},
		}, nil
	} else {
		h.logger.Error("request block not found", zap.String("key", req.UniqueKey))
		return nil, errors.New("")
	}
}

func (h *handler) StoreBlock(ctx context.Context, req *block_service.StoreBlockReq) (*block_service.StoreBlockResp, error) {
	err := h.fileStorage.Propose(filestore.StoreCommand, req.Entry)
	if err != nil {
		h.logger.Error("propose block in cluster error", zap.Error(err), zap.String("key", req.Entry.UniqueKey))
		return nil, err
	}
	return &block_service.StoreBlockResp{}, nil
}

func (h *handler) DeleteBlock(ctx context.Context, req *block_service.DeleteBlockReq) (*block_service.DeleteBlockResp, error) {
	err := h.fileStorage.Propose(filestore.DeleteCommand, &block_service.BlockEntry{UniqueKey: req.UniqueKey})
	if err != nil {
		h.logger.Error("remove block in cluster error", zap.Error(err), zap.String("key", req.UniqueKey))
		return nil, err
	}
	return &block_service.DeleteBlockResp{}, nil
}
