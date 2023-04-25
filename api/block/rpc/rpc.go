package rpc_api

import (
	"net"
	"net/url"

	"github.com/fooage/shamrock/core/filestore"
	"github.com/fooage/shamrock/core/raft"
	"github.com/fooage/shamrock/proto/proto_gen/block_service"
	"github.com/fooage/shamrock/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var server = grpc.NewServer()

func ServeRPC(logger *zap.Logger, local url.URL, fileStorage filestore.FileStorage, raftCluster raft.Cluster) {
	listener, err := net.Listen("tcp", utils.AddressOffsetRPC(local))
	if err != nil {
		logger.Panic("grpc listener init failed", zap.Error(err))
	}
	defer func() {
		if err := server.Serve(listener); err != nil {
			logger.Panic("grpc interface server run error", zap.Error(err))
		}
	}()

	// Register the relevant interface functions of the Block service.
	block_service.RegisterBlockServiceServer(server, generateHandler(logger, fileStorage, raftCluster))
}
