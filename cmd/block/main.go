package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"strings"

	http_api "github.com/fooage/shamrock/api/block/http"
	rpc_api "github.com/fooage/shamrock/api/block/rpc"
	"github.com/fooage/shamrock/core/filestore"
	"github.com/fooage/shamrock/core/raft"
	"github.com/fooage/shamrock/service"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.uber.org/zap"
)

// The block storage service is mainly responsible for the storage of object
// chunks in Shamrock. It is a very pure storage system, with chunk hash as key
// and file as value. There is no understanding of the meaning of the storage
// object and the meaning of chunk in the whole object.

var (
	local     *string
	group     *int
	self      *int
	peers     *string
	join      *bool
	discovery *string
)

// init load arguments in command line
func init() {
	local = flag.String("local", "127.0.0.1:12360", "the instance local address")
	group = flag.Int("group", 1, "group number for consensus")
	self = flag.Int("self", 1, "node number in the cluster")
	peers = flag.String("peers", "http://127.0.0.1:12360", "comma separated cluster peers")
	join = flag.Bool("join", false, "whether join an existing cluster")
	discovery = flag.String("discovery", "127.0.0.1:8500", "separated service discovery")
	flag.Parse()
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	// init two channels which connect layers
	proposeCh := make(chan string)
	defer close(proposeCh)
	confChangeCh := make(chan raftpb.ConfChange)
	defer close(confChangeCh)
	logger, _ := zap.NewProduction(zap.Fields(
		zap.Int("node", *self),
		zap.Int("group", *group),
	))

	// initialize the storage and consistency layers
	fileStorage := filestore.NewFileStoreServer(logger)
	raftCluster := raft.NewRaftServer(logger, *self, *group, strings.Split(*peers, ","), *join, proposeCh, confChangeCh, fileStorage.SnapshotFetch)
	fileStorage.Connect(raftCluster)

	// start block service's rpc apis
	local, _ := url.Parse(fmt.Sprintf("http://%s", *local))
	go http_api.ServeHttp(cancel, logger, *local, raftCluster)
	go rpc_api.ServeRPC(cancel, logger, *local, fileStorage, raftCluster)

	// register service to cluster service discovery
	local, _ = url.Parse(strings.Split(*peers, ",")[*self-1])
	discovery := service.InitServiceDiscovery(logger, "consul", strings.Split(*discovery, ","))
	discovery.Register("shamrock-block", *local)
	defer discovery.Deregister("shamrock-block")

	<-ctx.Done()
}
