package http_api

import (
	"context"
	"net/url"

	"github.com/fooage/shamrock/core/raft"
	"github.com/fooage/shamrock/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func ServeHttp(cancelFunc context.CancelFunc, logger *zap.Logger, local url.URL, raftCluster raft.Cluster) {
	router := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	handler := generateHandler(logger, raftCluster)
	defer func() {
		if err := router.Run(utils.AddressOffsetHTTP(local)); err != nil {
			logger.Panic("http interface router run error", zap.Error(err))
		}
		defer cancelFunc()
	}()

	// related to cluster configuration changes
	cluster := router.Group("/cluster/nodes")
	{
		cluster.POST("/:id", handler.ConfChangeAddNode)
		cluster.DELETE("/:id", handler.ConfChangeRemoveNode)
	}

	// service running status interface can be explored and queried
	service := router.Group("/service")
	{
		service.GET("/health", handler.QueryServiceHealth)
	}
}
