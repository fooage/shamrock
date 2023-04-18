package http_api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/fooage/shamrock/core/kvstore"
	"github.com/fooage/shamrock/core/raft"
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

func (h *handler) confChangeAddNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 0, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, nil)
		return
	}
	var requestBody struct {
		NodeUrl string `json:"node_url"`
	}
	err = c.BindJSON(&requestBody)
	if err != nil {
		c.JSON(http.StatusBadRequest, nil)
		return
	}
	// check url format avoid raft node crash
	_, err = url.ParseRequestURI(requestBody.NodeUrl)
	if err != nil {
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	h.raftCluster.ConfChange() <- raftpb.ConfChange{
		Type:    raftpb.ConfChangeAddNode,
		NodeID:  id,
		Context: []byte(requestBody.NodeUrl),
	}
	// As above, optimistic that raft will apply the config change.
	c.JSON(http.StatusAccepted, nil)
}

func (h *handler) confChangeRemoveNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 0, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	h.raftCluster.ConfChange() <- raftpb.ConfChange{
		Type:   raftpb.ConfChangeRemoveNode,
		NodeID: id,
	}
	c.JSON(http.StatusAccepted, nil)
}

func (h *handler) queryObjectMeta(c *gin.Context) {
	name := c.Param("name")
	key := generateObjectMetaKey(name)

	if value, ok := h.kvStorage.Lookup(key); ok {
		var data ObjectMeta
		err := json.Unmarshal([]byte(value), &data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
		c.JSON(http.StatusOK, data)
	} else {
		c.JSON(http.StatusNotFound, nil)
	}
}

func (h *handler) updateObjectMeta(c *gin.Context) {
	name := c.Param("name")
	key := generateObjectMetaKey(name)
	data := ObjectMeta{}
	err := c.BindJSON(&data)
	if err != nil {
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	value, err := json.Marshal(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, nil)
		return
	}
	if err = h.kvStorage.Propose(key, string(value)); err != nil {
		c.JSON(http.StatusInternalServerError, nil)
		return
	}

	// Not waiting for ack from raft. Value is not yet committed so a
	// subsequent GET on the key may return old value.
	c.JSON(http.StatusAccepted, nil)
}

func (h *handler) queryChunkMetas(c *gin.Context) {
	var requestBody struct {
		HashList  []string `json:"hash_list"`
		BatchSize int      `json:"batch_size"`
	}
	err := c.BindJSON(&requestBody)
	if err != nil || requestBody.BatchSize > 100 || requestBody.BatchSize != len(requestBody.HashList) {
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	result := make([]ChunkMeta, 0, requestBody.BatchSize)
	for _, hash := range requestBody.HashList {
		key := generateChunkMetaKey(hash)
		if value, ok := h.kvStorage.Lookup(key); ok {
			var data ChunkMeta
			err := json.Unmarshal([]byte(value), &data)
			if err != nil {
				c.JSON(http.StatusInternalServerError, nil)
				return
			}
			result = append(result, data)
		}
	}

	// This query may not return all chunk metas needed, but think it is acceptable.
	if len(result) < requestBody.BatchSize {
		c.JSON(http.StatusNotFound, result)
	} else {
		c.JSON(http.StatusOK, result)
	}
}

func (h *handler) queryChunkMeta(c *gin.Context) {
	hash := c.Param("hash")
	key := generateChunkMetaKey(hash)

	if value, ok := h.kvStorage.Lookup(key); ok {
		var data ChunkMeta
		err := json.Unmarshal([]byte(value), &data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, nil)
			return
		}
		c.JSON(http.StatusOK, data)
	} else {
		c.JSON(http.StatusNotFound, nil)
	}
}

func (h *handler) updateChunkMeta(c *gin.Context) {
	name := c.Param("hash")
	key := generateChunkMetaKey(name)
	data := ChunkMeta{}
	err := c.BindJSON(&data)
	if err != nil {
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	value, err := json.Marshal(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, nil)
		return
	}
	if err = h.kvStorage.Propose(key, string(value)); err != nil {
		c.JSON(http.StatusInternalServerError, nil)
		return
	}

	// Not waiting for ack from raft. Value is not yet committed so a
	// subsequent GET on the key may return old value.
	c.JSON(http.StatusAccepted, nil)
}
