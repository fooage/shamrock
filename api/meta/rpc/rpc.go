package rpc_api

import (
	"context"
	"net"
	"net/url"

	"github.com/fooage/shamrock/core/kvstore"
	"github.com/fooage/shamrock/core/raft"
	"github.com/fooage/shamrock/proto/proto_gen/meta_service"
	"github.com/fooage/shamrock/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var server = grpc.NewServer()

func ServeRPC(cancelFunc context.CancelFunc, logger *zap.Logger, local url.URL, kvStorage kvstore.KVStorage, raftCluster raft.Cluster) {
	listener, err := net.Listen("tcp", utils.AddressOffsetRPC(local))
	if err != nil {
		logger.Panic("grpc listener init failed", zap.Error(err))
	}
	defer func() {
		if err := server.Serve(listener); err != nil {
			logger.Panic("grpc interface server run error", zap.Error(err))
		}
		defer cancelFunc()
	}()

	// Register the relevant interface functions of the Block service.
	meta_service.RegisterMetaServiceServer(server, generateHandler(logger, kvStorage, raftCluster))
}
