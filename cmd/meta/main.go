package main

import (
	"flag"
	"net/url"
	"strings"

	http_api "github.com/fooage/shamrock/api/meta/http"
	rpc_api "github.com/fooage/shamrock/api/meta/rpc"
	"github.com/fooage/shamrock/core/kvstore"
	"github.com/fooage/shamrock/core/raft"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.uber.org/zap"
)

// Meta-Server is responsible for storing Block-Server cluster and group
// information, and storing the mapping relationship between file hash and
// slice hash. And it can return the routing information of the slices of the
// entire storage cluster to the client.

var (
	group *int
	self  *int
	peers *string
	join  *bool
)

// init load arguments in command line
func init() {
	group = flag.Int("group", 1, "group number for consensus")
	self = flag.Int("self", 1, "node number in the cluster")
	peers = flag.String("peers", "http://127.0.0.1:12379", "comma separated cluster peers")
	join = flag.Bool("join", false, "whether join an existing cluster")
	flag.Parse()
}

func main() {
	// init two channels which connect layers
	proposeCh := make(chan string)
	defer close(proposeCh)
	confChangeCh := make(chan raftpb.ConfChange)
	defer close(confChangeCh)
	logger, _ := zap.NewProduction(zap.Fields(
		zap.Int("node", *self),
		zap.Int("group", *group)),
	)

	// initialize the storage and consistency layers
	kvStorage := kvstore.NewKVStoreServer(logger)
	raftCluster := raft.NewRaftServer(logger, *self, *group, strings.Split(*peers, ","), *join, proposeCh, confChangeCh, kvStorage.SnapshotFetch)
	kvStorage.Connect(raftCluster)

	// the key-value http handler will propose updates to raft
	local, _ := url.Parse(strings.Split(*peers, ",")[*self-1])
	go http_api.ServeHttp(logger, *local, kvStorage, raftCluster)
	rpc_api.ServeRPC(logger, *local, kvStorage, raftCluster)
}
