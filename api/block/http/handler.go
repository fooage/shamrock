package http_api

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/fooage/shamrock/core/raft"
	"github.com/fooage/shamrock/proto/proto_gen/block_service"
	"github.com/fooage/shamrock/utils"
	"github.com/gin-gonic/gin"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.uber.org/zap"
)

type handler struct {
	raftCluster raft.Cluster
	logger      *zap.Logger
}

func generateHandler(logger *zap.Logger, raftCluster raft.Cluster) *handler {
	return &handler{
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

func (h *handler) QueryServiceHealth(c *gin.Context) {
	capacityUsed, capacity, err := utils.DiskUsage("/")
	if err != nil {
		h.logger.Error("disk usage get failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, nil)
		return
	}
	c.JSON(http.StatusOK, block_service.HealthReport{
		Address:      h.raftCluster.Current()[0].String(),
		StoreGroup:   int64(h.raftCluster.Group()),
		StoreNode:    int64(h.raftCluster.Self()),
		Capacity:     int64(capacity),
		CapacityUsed: int64(capacityUsed),
	})
}
