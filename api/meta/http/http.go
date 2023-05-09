// This is a generic node API that follows the REST API style, which can
// provide operation interfaces for upper-layer Meta service and Block
// service's KV storage.

package http_api

import (
	"net/url"

	"github.com/fooage/shamrock/core/kvstore"
	"github.com/fooage/shamrock/core/raft"
	"github.com/fooage/shamrock/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func ServeHttp(logger *zap.Logger, local url.URL, kvStorage kvstore.KVStorage, raftCluster raft.Cluster) {
	router := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	handler := generateHandler(logger, kvStorage, raftCluster)
	defer func() {
		if err := router.Run(utils.AddressOffsetHTTP(local)); err != nil {
			logger.Panic("http interface router run error", zap.Error(err))
		}
	}()

	// related to cluster configuration changes
	cluster := router.Group("/cluster/nodes")
	{
		cluster.POST("/:id", handler.ConfChangeAddNode)
		cluster.DELETE("/:id", handler.ConfChangeRemoveNode)
	}

	// object meta info storage-related operational apis
	object := router.Group("/meta/objects")
	{
		object.GET("/:unique_key", handler.QueryObjectMeta)
		object.PUT("/:unique_key", handler.UpdateObjectMeta)
	}

	// chunk meta info related apis for route client request
	chunk := router.Group("/meta/chunks")
	{
		chunk.GET("/:unique_key", handler.QueryChunkMeta)
		chunk.PUT("/:unique_key", handler.UpdateChunkMeta)
	}
}
