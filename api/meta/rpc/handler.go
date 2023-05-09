package rpc_api

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/fooage/shamrock/core/kvstore"
	"github.com/fooage/shamrock/core/raft"
	"github.com/fooage/shamrock/proto/proto_gen/meta_service"
	"go.uber.org/zap"
)

type handler struct {
	kvStorage   kvstore.KVStorage
	raftCluster raft.Cluster
	logger      *zap.Logger
	// implement abstract meta service method
	meta_service.UnimplementedMetaServiceServer
}

func generateHandler(logger *zap.Logger, kvStorage kvstore.KVStorage, raftCluster raft.Cluster) *handler {
	return &handler{
		kvStorage:   kvStorage,
		raftCluster: raftCluster,
		logger:      logger,
	}
}

func (h *handler) QueryObjectMeta(ctx context.Context, req *meta_service.QueryObjectMetaReq) (*meta_service.QueryObjectMetaResp, error) {
	key := kvstore.GenerateObjectMetaKey(req.UniqueKey)
	value, ok := h.kvStorage.Lookup(key)
	if ok {
		meta := meta_service.ObjectMeta{}
		err := json.Unmarshal([]byte(value), &meta)
		if err != nil {
			h.logger.Error("object meta unmarshal failed", zap.Error(err), zap.String("key", req.GetUniqueKey()))
			return nil, err
		}
		return &meta_service.QueryObjectMetaResp{
			Meta: &meta,
		}, nil
	}
	return nil, errors.New("query object not found")
}

func (h *handler) UpdateObjectStatus(ctx context.Context, req *meta_service.UpdateObjectStatusReq) (*meta_service.UpdateObjectStatusResp, error) {
	key := kvstore.GenerateObjectMetaKey(req.UniqueKey)
	value, ok := h.kvStorage.Lookup(key)
	if ok {
		meta := meta_service.ObjectMeta{}
		err := json.Unmarshal([]byte(value), &meta)
		if err != nil {
			h.logger.Error("object meta unmarshal failed", zap.Error(err), zap.String("key", req.GetUniqueKey()))
			return nil, err
		}
		meta.Status = req.Status
		value, err := json.Marshal(meta)
		err = h.kvStorage.Propose(key, string(value))
		if err != nil {
			h.logger.Error("update object status error", zap.Error(err))
			return nil, err
		}
		return &meta_service.UpdateObjectStatusResp{}, nil
	}
	return nil, errors.New("query object not found")
}

func (h *handler) QueryChunkMeta(ctx context.Context, req *meta_service.QueryChunkMetaReq) (*meta_service.QueryChunkMetaResp, error) {
	if len(req.UniqueKeys) > 1000 || req.UniqueKeys == nil {
		return nil, errors.New("query list over limit")
	}

	result := make(map[string]*meta_service.ChunkMeta, len(req.UniqueKeys))
	for _, uniqueKey := range req.UniqueKeys {
		key := kvstore.GenerateChunkMetaKey(uniqueKey)
		value, ok := h.kvStorage.Lookup(key)
		if !ok {
			h.logger.Info("query chunk not found", zap.String("key", uniqueKey))
			continue
		}
		result[uniqueKey] = &meta_service.ChunkMeta{}
		err := json.Unmarshal([]byte(value), result[uniqueKey])
		if err != nil {
			h.logger.Error("object meta unmarshal failed", zap.Error(err), zap.String("key", uniqueKey))
			continue
		}
	}
	return &meta_service.QueryChunkMetaResp{Result: result}, nil
}

func (h *handler) UpdateChunkStatus(ctx context.Context, req *meta_service.UpdateChunkStatusReq) (*meta_service.UpdateChunkStatusResp, error) {
	if len(req.UniqueKeys) > 1000 || req.UniqueKeys == nil {
		return nil, errors.New("param list over limit")
	}

	for _, uniqueKey := range req.UniqueKeys {
		key := kvstore.GenerateChunkMetaKey(uniqueKey)
		data, ok := h.kvStorage.Lookup(key)
		if !ok {
			h.logger.Info("query chunk not found", zap.String("key", uniqueKey))
			continue
		}
		meta := meta_service.ChunkMeta{}
		err := json.Unmarshal([]byte(data), &meta)
		if err != nil {
			h.logger.Error("object meta unmarshal failed", zap.Error(err), zap.String("key", uniqueKey))
			return nil, err
		}
		meta.Status = req.Status
		value, err := json.Marshal(meta)
		err = h.kvStorage.Propose(key, string(value))
		if err != nil {
			h.logger.Error("update object status error", zap.Error(err))
			return nil, err
		}
	}
	return &meta_service.UpdateChunkStatusResp{}, nil
}

func (h *handler) RegisterObject(ctx context.Context, req *meta_service.RegisterObjectReq) (*meta_service.RegisterObjectResp, error) {
	panic("implement me")
}
