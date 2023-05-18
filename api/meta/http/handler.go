package http_api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/fooage/shamrock/core/kvstore"
	"github.com/fooage/shamrock/core/raft"
	"github.com/fooage/shamrock/proto/proto_gen/meta_service"
	"github.com/gin-gonic/gin"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.uber.org/zap"
)

type handler struct {
	kvStorage   kvstore.KVStorage
	raftCluster raft.Cluster
	logger      *zap.Logger
}

func generateHandler(logger *zap.Logger, kvStorage kvstore.KVStorage, raftCluster raft.Cluster) *handler {
	return &handler{
		kvStorage:   kvStorage,
		raftCluster: raftCluster,
		logger:      logger,
	}
}

func (h *handler) ConfChangeAddNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 0, 64)
	if err != nil || id == 0 {
		h.logger.Error("parse url param error", zap.Error(err), zap.Uint64("id", id))
		c.JSON(http.StatusBadRequest, nil)
		return
	}
	var reqBody struct {
		NodeUrl string `json:"node_url"`
	}
	err = c.BindJSON(&reqBody)
	if err != nil {
		h.logger.Error("bind json param error", zap.Error(err))
		c.JSON(http.StatusBadRequest, nil)
		return
	}
	// check url format avoid raft node crash
	_, err = url.ParseRequestURI(reqBody.NodeUrl)
	if err != nil {
		h.logger.Error("new node url can not parse", zap.Error(err))
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	h.raftCluster.ConfChange() <- raftpb.ConfChange{
		Type:    raftpb.ConfChangeAddNode,
		NodeID:  id,
		Context: []byte(reqBody.NodeUrl),
	}
	// As above, optimistic that raft will apply the config change.
	c.JSON(http.StatusAccepted, nil)
}

func (h *handler) ConfChangeRemoveNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 0, 64)
	if err != nil || id == 0 {
		h.logger.Error("parse url param error", zap.Error(err), zap.Uint64("id", id))
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	h.raftCluster.ConfChange() <- raftpb.ConfChange{
		Type:   raftpb.ConfChangeRemoveNode,
		NodeID: id,
	}
	c.JSON(http.StatusAccepted, nil)
}

func (h *handler) QueryObjectMeta(c *gin.Context) {
	uniqueKey := c.Param("unique_key")
	if value, ok := h.kvStorage.Lookup(uniqueKey); ok {
		var data meta_service.ObjectMeta
		err := json.Unmarshal([]byte(value), &data)
		if err != nil {
			h.logger.Error("object meta json unmarshal failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
		c.JSON(http.StatusOK, data)
	} else {
		c.JSON(http.StatusNotFound, nil)
	}
}

func (h *handler) UpdateObjectMeta(c *gin.Context) {
	uniqueKey := c.Param("unique_key")
	data := meta_service.ObjectMeta{}
	err := c.BindJSON(&data)
	if err != nil {
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	value, err := json.Marshal(data)
	if err != nil {
		h.logger.Error("object meta json marshal failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, nil)
		return
	}
	if err = h.kvStorage.Propose(uniqueKey, string(value)); err != nil {
		h.logger.Error("propose raft layer error", zap.Error(err), zap.String("key", uniqueKey))
		c.JSON(http.StatusInternalServerError, nil)
		return
	}

	// At present, submitting data is a reliable synchronization process, and it
	// can return success only if it is accepted by more than half of the nodes.
	c.JSON(http.StatusOK, nil)
}

func (h *handler) QueryChunkMeta(c *gin.Context) {
	uniqueKey := c.Param("unique_key")
	if value, ok := h.kvStorage.Lookup(uniqueKey); ok {
		var data meta_service.ChunkMeta
		err := json.Unmarshal([]byte(value), &data)
		if err != nil {
			h.logger.Error("chunk meta json unmarshal failed", zap.Error(err))
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
		c.JSON(http.StatusOK, data)
	} else {
		c.JSON(http.StatusNotFound, nil)
	}
}

func (h *handler) UpdateChunkMeta(c *gin.Context) {
	uniqueKey := c.Param("unique_key")
	data := meta_service.ChunkMeta{}
	err := c.BindJSON(&data)
	if err != nil {
		h.logger.Error("bind json param error", zap.Error(err))
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	value, err := json.Marshal(data)
	if err != nil {
		h.logger.Error("chunk meta json marshal failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, nil)
		return
	}
	if err = h.kvStorage.Propose(uniqueKey, string(value)); err != nil {
		h.logger.Error("propose raft layer error", zap.Error(err), zap.String("key", uniqueKey))
		c.JSON(http.StatusInternalServerError, nil)
		return
	}

	// At present, submitting data is a reliable synchronization process, and it
	// can return success only if it is accepted by more than half of the nodes.
	c.JSON(http.StatusOK, nil)
}

func (h *handler) QueryServiceHealth(c *gin.Context) {
	// TODO: After that, it can make some reports on the service status.
	c.JSON(http.StatusOK, nil)
}
