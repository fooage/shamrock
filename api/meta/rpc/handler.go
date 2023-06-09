package rpc_api

import (
	"context"
	"encoding/json"
	"errors"
	"math"

	"github.com/fooage/shamrock/core/filestore"
	"github.com/fooage/shamrock/core/kvstore"
	"github.com/fooage/shamrock/core/raft"
	"github.com/fooage/shamrock/core/scheduler"
	"github.com/fooage/shamrock/proto/proto_gen/meta_service"
	"go.uber.org/zap"
)

type handler struct {
	kvStorage      kvstore.KVStorage
	raftCluster    raft.Cluster
	blockScheduler scheduler.Scheduler
	logger         *zap.Logger
	// implement abstract meta service method
	meta_service.UnimplementedMetaServiceServer
}

func generateHandler(logger *zap.Logger, kvStorage kvstore.KVStorage, raftCluster raft.Cluster, blockScheduler scheduler.Scheduler) *handler {
	return &handler{
		kvStorage:      kvStorage,
		raftCluster:    raftCluster,
		blockScheduler: blockScheduler,
		logger:         logger,
	}
}

func (h *handler) QueryObjectMeta(ctx context.Context, req *meta_service.QueryObjectMetaReq) (*meta_service.QueryObjectMetaResp, error) {
	value, ok := h.kvStorage.Lookup(req.UniqueKey)
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

func (h *handler) QueryObjectKeys(ctx context.Context, req *meta_service.QueryObjectKeysReq) (*meta_service.QueryObjectKeysResp, error) {
	begin := int(req.Page * req.PageSize)
	end := int((req.Page + 1) * req.PageSize)
	matched := h.kvStorage.Match(req.Prefix)
	if len(matched) == 0 {
		h.logger.Error("prefix not match object key", zap.String("prefix", req.Prefix))
		return nil, errors.New("object keys not found")
	}
	end = int(math.Min(float64(end), float64(len(matched))))
	if begin < 0 || begin > len(matched) || begin >= end {
		h.logger.Error("page params error", zap.Int64("page", req.Page), zap.Int64("page_size", req.PageSize))
		return nil, errors.New("params invalid")
	}
	return &meta_service.QueryObjectKeysResp{
		UniqueKeys: matched[begin:end],
		Total:      int64(len(matched)),
	}, nil
}

func (h *handler) UpdateObjectStatus(ctx context.Context, req *meta_service.UpdateObjectStatusReq) (*meta_service.UpdateObjectStatusResp, error) {
	value, ok := h.kvStorage.Lookup(req.UniqueKey)
	if ok {
		meta := meta_service.ObjectMeta{}
		err := json.Unmarshal([]byte(value), &meta)
		if err != nil {
			h.logger.Error("object meta unmarshal failed", zap.Error(err), zap.String("key", req.GetUniqueKey()))
			return nil, err
		}
		meta.Status = req.Status
		value, err := json.Marshal(meta)
		err = h.kvStorage.Propose(req.UniqueKey, string(value))
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
		value, ok := h.kvStorage.Lookup(uniqueKey)
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
		data, ok := h.kvStorage.Lookup(uniqueKey)
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
		err = h.kvStorage.Propose(uniqueKey, string(value))
		if err != nil {
			h.logger.Error("update object status error", zap.Error(err))
			return nil, err
		}
	}
	return &meta_service.UpdateChunkStatusResp{}, nil
}

func (h *handler) RegisterObject(ctx context.Context, req *meta_service.RegisterObjectReq) (*meta_service.RegisterObjectResp, error) {
	object := meta_service.ObjectMeta{
		UniqueKey: kvstore.GenerateObjectKey(req.Name),
		Status:    meta_service.EntryStatus_Registered,
		Size:      req.Size,
		ChunkList: make([]string, 0, len(req.HashList)),
	}

	// Generate meta information for each chunk and dispatch storage group.
	// Finally complete the registration and return of object meta information.
	for index, hash := range req.HashList {
		storeGroup, err := h.blockScheduler.Dispatch(filestore.DefaultBlockSize)
		if err != nil {
			h.logger.Error("scheduler dispatch chunk error", zap.Error(err), zap.String("file", req.Name))
			return nil, err
		}
		chunk := meta_service.ChunkMeta{
			UniqueKey:  kvstore.GenerateChunkKey(hash),
			Status:     meta_service.EntryStatus_Registered,
			Parent:     object.UniqueKey,
			Index:      int64(index),
			StoreGroup: storeGroup,
		}
		chunkJson, _ := json.Marshal(chunk)
		err = h.kvStorage.Propose(chunk.UniqueKey, string(chunkJson))
		if err != nil {
			h.logger.Error("register chunk meta error", zap.Error(err), zap.String("key", chunk.UniqueKey))
			return nil, err
		}
		// chunk unique key is added to the object meta
		object.ChunkList = append(object.ChunkList, chunk.UniqueKey)
	}

	objectJson, _ := json.Marshal(object)
	err := h.kvStorage.Propose(object.UniqueKey, string(objectJson))
	if err != nil {
		h.logger.Error("register object meta error", zap.Error(err))
		return nil, err
	}
	return &meta_service.RegisterObjectResp{Meta: &object}, nil
}

func (h *handler) QueryStorageAddress(ctx context.Context, req *meta_service.QueryStorageAddressReq) (*meta_service.QueryStorageAddressResp, error) {
	addressList, err := h.blockScheduler.Proxy(req.StoreGroup, req.FromMaster)
	if err != nil {
		h.logger.Error("scheduler query node error", zap.Error(err))
		return nil, err
	}
	return &meta_service.QueryStorageAddressResp{
		AddressList: addressList,
	}, nil
}
