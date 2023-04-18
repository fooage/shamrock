// This is a generic node API that follows the REST API style, which can
// provide operation interfaces for upper-layer Meta service and Block
// service's KV storage.

package http_api

import (
	"fmt"

	"github.com/fooage/shamrock/core/kvstore"
	"github.com/fooage/shamrock/core/raft"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func ServeHttp(logger *zap.Logger, port int, kvStorage kvstore.KVStorage, raftCluster raft.Cluster) {
	router := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	handler := generateHandler(logger, kvStorage, raftCluster)
	defer func() {
		if err := router.Run(fmt.Sprintf(":%d", port)); err != nil {
			logger.Panic("http interface router run error", zap.Error(err))
		}
	}()

	// related to cluster configuration changes
	cluster := router.Group("/cluster/nodes")
	{
		cluster.POST("/:id", handler.confChangeAddNode)
		cluster.DELETE("/:id", handler.confChangeRemoveNode)
	}

	// object meta info storage-related operational apis
	object := router.Group("/meta/objects")
	{
		object.GET("/:name", handler.queryObjectMeta)
		object.PUT("/:name", handler.updateObjectMeta)
	}

	// chunk meta info related apis for route client request
	chunk := router.Group("/meta/chunks")
	{
		chunk.GET("/", handler.queryChunkMetas)
		chunk.GET("/:hash", handler.queryChunkMeta)
		chunk.PUT("/:hash", handler.updateChunkMeta)
	}
}
