package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"strings"

	http_api "github.com/fooage/shamrock/api/meta/http"
	rpc_api "github.com/fooage/shamrock/api/meta/rpc"
	"github.com/fooage/shamrock/core/kvstore"
	"github.com/fooage/shamrock/core/raft"
	"github.com/fooage/shamrock/core/scheduler"
	"github.com/fooage/shamrock/service"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.uber.org/zap"
)

// Meta-Server is responsible for storing Block-Server cluster and group
// information, and storing the mapping relationship between file hash and
// slice hash. And it can return the routing information of the slices of the
// entire storage cluster to the client.

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
		zap.Int("group", *group)),
	)

	// initialize the storage, consistency layers and scheduler
	discovery := service.InitServiceDiscovery(logger, "consul", strings.Split(*discovery, ","))
	kvStorage := kvstore.NewKVStoreServer(logger)
	raftCluster := raft.NewRaftServer(logger, *self, *group, strings.Split(*peers, ","), *join, proposeCh, confChangeCh, kvStorage.SnapshotFetch)
	blockScheduler := scheduler.NewScheduler(logger)
	kvStorage.Connect(raftCluster)

	// the key-value http handler will propose updates to raft
	local, _ := url.Parse(fmt.Sprintf("http://%s", *local))
	go http_api.ServeHttp(cancel, logger, *local, kvStorage, raftCluster)
	go rpc_api.ServeRPC(cancel, logger, *local, kvStorage, raftCluster, blockScheduler)

	// register service to cluster service discovery
	local, _ = url.Parse(strings.Split(*peers, ",")[*self-1])
	discovery.Register("shamrock-meta", *local)
	defer discovery.Deregister("shamrock-meta")

	<-ctx.Done()
}
