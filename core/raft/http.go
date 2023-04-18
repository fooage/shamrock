package raft

import (
	"context"

	"go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
)

// These methods implement the methods defined by the 'rafthttp.Raft' interface.
// The raft module does not provide the relevant implementation of the network
// layer, but encapsulates the message to be sent into the Ready struct and
// returns it to the upper module, and then the upper module decides how to send
// these messages to other nodes in the cluster.

func (rf *raftServer) Process(ctx context.Context, m raftpb.Message) error {
	return rf.node.Step(ctx, m)
}

func (rf *raftServer) IsIDRemoved(id uint64) bool {
	return false
}

func (rf *raftServer) ReportUnreachable(id uint64) {
	rf.node.ReportUnreachable(id)
}

func (rf *raftServer) ReportSnapshot(id uint64, status raft.SnapshotStatus) {
	rf.node.ReportSnapshot(id, status)
}
