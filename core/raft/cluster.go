package raft

import (
	"net/url"
	"sync"

	"go.etcd.io/etcd/client/pkg/v3/types"
	"go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
)

// Cluster abstracts the cluster concept and provides methods for
// upper-level calls, which is Raft's interface to upper-layer. The whole
// cluster may be separated and used as an independent class.
type Cluster interface {
	Commit() <-chan *Commit
	Error() <-chan error
	Snapshot() <-chan *snap.Snapshotter
	Propose() chan<- string
	ConfChange() chan<- raftpb.ConfChange
	Self() types.ID
	Group() types.ID
	Leader() (types.URLs, bool)
}

func (rf *raftServer) Commit() <-chan *Commit {
	return rf.commitCh
}

func (rf *raftServer) Error() <-chan error {
	return rf.errorCh
}

func (rf *raftServer) Snapshot() <-chan *snap.Snapshotter {
	return rf.snapshotReadyCh
}

func (rf *raftServer) Propose() chan<- string {
	return rf.proposeCh
}

func (rf *raftServer) ConfChange() chan<- raftpb.ConfChange {
	return rf.confChangeCh
}

func (rf *raftServer) Leader() (types.URLs, bool) {
	for _, peer := range rf.cluster.peers {
		if peer.status == raft.StateLeader {
			return peer.urls, true
		}
	}
	return nil, false
}

func (rf *raftServer) Self() types.ID {
	return types.ID(rf.cluster.self)
}

func (rf *raftServer) Group() types.ID {
	return types.ID(rf.cluster.group)
}

type cluster struct {
	self  int               // number of the current node in the single raft cluster
	group int               // number of the cluster where the node resides in the Multi-Raft
	peers map[types.ID]node // peer store information about all nodes in the cluster
	mutex sync.Mutex
}

type node struct {
	urls   types.URLs
	status raft.StateType
}

func (c *cluster) join(id types.ID, raws []string) {
	if len(raws) == 0 {
		panic("empty node join cluster")
	}
	var urls types.URLs
	for _, raw := range raws {
		result, err := url.Parse(raw)
		if err != nil {
			panic(err)
		}
		urls = append(urls, *result)
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	// TODO: It may be necessary to check whether the node state is actually set follower.
	c.peers[id] = node{urls: urls, status: raft.StateFollower}
}

func (c *cluster) update(id types.ID, status raft.StateType) {
	c.mutex.Lock()
	if peer, ok := c.peers[id]; ok {
		peer.status = status
		c.peers[id] = peer
	}
	c.mutex.Unlock()
}

func (c *cluster) leave(id types.ID) {
	c.mutex.Lock()
	delete(c.peers, id)
	c.mutex.Unlock()
}
