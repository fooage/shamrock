package main

import (
	"flag"
	"strings"

	http_api "github.com/fooage/shamrock/api/meta/http"
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
	port  *int
	group *int
	self  *int
	peers *string
	join  *bool
)

// init load arguments in command line
func init() {
	port = flag.Int("port", 10101, "http api server port")
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

	// kv provide a storage for meta info which store in this server
	kvStorage := kvstore.NewKVStoreServer(logger)
	// raft provides a commit stream for the proposals from the http api
	raftCluster := raft.NewRaftServer(logger, *self, *group, strings.Split(*peers, ","), *join, proposeCh, confChangeCh, kvStorage.SnapshotFetch)
	kvStorage.Connect(raftCluster)

	// the key-value http handler will propose updates to raft
	http_api.ServeHttp(logger, *port, kvStorage, raftCluster)
}
