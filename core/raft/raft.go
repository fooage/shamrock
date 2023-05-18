package raft

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/fooage/shamrock/utils"
	"go.etcd.io/etcd/client/pkg/v3/types"
	"go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.etcd.io/etcd/server/v3/etcdserver/api/rafthttp"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
	stats "go.etcd.io/etcd/server/v3/etcdserver/api/v2stats"
	"go.etcd.io/etcd/server/v3/wal"
	"go.uber.org/zap"
)

type raftServer struct {
	// This channel for the application to pass the customer's update request
	// to the underlying raft layer. There is also a channel dedicated to
	// passing cluster configuration changes.
	proposeCh    chan string
	confChangeCh chan raftpb.ConfChange

	// The channel through which the underlying raft layer notifies the
	// application of the instructions log could be committed.
	commitCh chan *Commit
	errorCh  chan error

	node     raft.Node
	cluster  cluster
	walPath  string // storage path for WAL files
	snapPath string // path for snapshot directory

	stableStorage   *wal.WAL
	unstableStorage *raft.MemoryStorage
	confState       raftpb.ConfState
	appliedIndex    uint64
	snapshot        *snap.Snapshotter
	snapshotReadyCh chan *snap.Snapshotter
	snapshotFetch   func() ([]byte, error)
	snapshotIndex   uint64

	// Network transport between raft node, and some channel responsibility for
	// Application layer stop and Transport layer stop.
	transport  *rafthttp.Transport
	appStopCh  chan struct{}
	httpStopCh chan struct{}
	httpDoneCh chan struct{}
	logger     *zap.Logger
}

// The Commit data structure submitted by application layer within one time will have a
// channel to notify whether it is applied.
type Commit struct {
	Data        []string
	ApplyDoneCh chan<- struct{}
}

var defaultSnapshotLimit uint64 = 2048
var defaultCompactLimit uint64 = 2048

// NewRaftServer initiates a raft instance and returns a committed log entry
// channel, error channel and a ready snapshot.
func NewRaftServer(logger *zap.Logger, self int, group int, peers []string, isJoin bool, proposeCh chan string, confChangeCh chan raftpb.ConfChange, snapshotFetch func() ([]byte, error)) Cluster {
	ins := &raftServer{
		cluster:         cluster{self: self, group: group, peers: make(map[types.ID]node)},
		proposeCh:       proposeCh,
		confChangeCh:    confChangeCh,
		commitCh:        make(chan *Commit),
		errorCh:         make(chan error),
		logger:          logger,
		walPath:         fmt.Sprintf("./storage/raft-%d-%d", group, self),
		snapPath:        fmt.Sprintf("./storage/raft-%d-%d-snap", group, self),
		snapshotFetch:   snapshotFetch,
		appStopCh:       make(chan struct{}),
		httpStopCh:      make(chan struct{}),
		httpDoneCh:      make(chan struct{}),
		snapshotReadyCh: make(chan *snap.Snapshotter, 1),
		// rest of structure init after WAL replay
	}

	go ins.startService(peers, isJoin)
	return ins
}

// startService as a goroutine to running raft state machine.
func (rf *raftServer) startService(initPeers []string, isJoin bool) {
	if !utils.PathExist(rf.snapPath) {
		if err := os.MkdirAll(rf.snapPath, 0750); err != nil {
			rf.logger.Panic("create snapshot folder error", zap.String("path", rf.snapPath), zap.Error(err))
		}
	}

	// signal WAL replay has finished and snapshot ready
	isLogExist := wal.Exist(rf.walPath)
	rf.snapshot = snap.New(rf.logger, rf.snapPath)
	rf.stableStorage = rf.replayWAL()
	rf.snapshotReadyCh <- rf.snapshot

	// loading config and peers in raft cluster
	config := &raft.Config{
		ID:                        uint64(rf.cluster.self),
		ElectionTick:              10,
		HeartbeatTick:             1,
		Storage:                   rf.unstableStorage,
		MaxSizePerMsg:             1024 * 1024,
		MaxInflightMsgs:           256,
		MaxUncommittedEntriesSize: 1 << 10,
	}
	peers := make([]raft.Peer, len(initPeers))
	for i := range peers {
		peers[i] = raft.Peer{ID: uint64(i + 1)}
	}
	if isJoin || isLogExist {
		rf.logger.Info("restart node and join group")
		rf.node = raft.RestartNode(config)
	} else {
		rf.logger.Info("start node as new one in group")
		rf.node = raft.StartNode(config, peers)
	}

	// Start the network layer of the Raft algorithm, responsible for network
	// transport between Raft nodes in the cluster.
	rf.transport = &rafthttp.Transport{
		Logger:      rf.logger,
		ID:          types.ID(rf.cluster.self),
		ClusterID:   types.ID(rf.cluster.group),
		Raft:        rf,
		ServerStats: stats.NewServerStats("", ""),
		LeaderStats: stats.NewLeaderStats(rf.logger, strconv.Itoa(rf.cluster.self)),
		ErrorC:      make(chan error),
	}

	if err := rf.transport.Start(); err != nil {
		rf.logger.Panic("http transport start error", zap.Error(err))
	}
	for i, peer := range initPeers {
		rf.cluster.join(types.ID(i+1), []string{peer})
		if i+1 != rf.cluster.self {
			rf.transport.AddPeer(types.ID(i+1), []string{peer})
			rf.logger.Info("loading peer in same group", zap.String("peer", peer))
		}
	}

	// Run goroutine to serve raft state machine.
	go rf.serveTransport()
	go rf.serveRequest()
}

func (rf *raftServer) serveTransport() {
	defer close(rf.httpDoneCh)
	self := rf.cluster.peers[types.ID(rf.cluster.self)]
	address := fmt.Sprintf("0.0.0.0:%s", self.urls[0].Port())
	listener, err := newStoppableListener(rf.logger, address, rf.httpStopCh)
	if err != nil {
		rf.logger.Panic("create raft listener error", zap.String("host", self.urls[0].Host), zap.Error(err))
	}

	// Serve accepts incoming connections on the listener, creating a
	// new service goroutine for each.
	rf.logger.Info("node begin to listen others")
	err = (&http.Server{Handler: rf.transport.Handler()}).Serve(listener)
	select {
	case <-rf.httpStopCh:
	default:
		rf.logger.Panic("transport failed to serve raft and close http server")
	}
}

func (rf *raftServer) serveRequest() {
	snapshot, err := rf.unstableStorage.Snapshot()
	if err != nil {
		rf.logger.Panic("unstable storage snapshot load error", zap.Error(err))
	}
	rf.confState = snapshot.Metadata.ConfState
	rf.snapshotIndex = snapshot.Metadata.Index
	rf.appliedIndex = snapshot.Metadata.Index

	go rf.applicationHandler()
	go rf.consistencyHandler()
}

// applicationHandler handle data and cluster configuration changes passed by
// the application layer, as a middle layer between them.
func (rf *raftServer) applicationHandler() {
	confChangeCount := uint64(0)

	for rf.proposeCh != nil && rf.confChangeCh != nil {
		select {
		case propose, ok := <-rf.proposeCh:
			if ok {
				// Wait propose accepted by more than half raft state
				// machine in cluster.
				err := rf.node.Propose(context.Background(), []byte(propose))
				if err != nil {
					rf.logger.Error("raft layer propose error", zap.Error(err))
				}
			} else {
				rf.proposeCh = nil
			}

		case confChange, ok := <-rf.confChangeCh:
			if ok {
				// Wait propose config change accepted by more than half
				// raft state machine in cluster.
				confChangeCount++
				confChange.ID = confChangeCount
				err := rf.node.ProposeConfChange(context.TODO(), confChange)
				if err != nil {
					rf.logger.Error("raft layer propose config error", zap.Error(err))
				}
			} else {
				rf.confChangeCh = nil
			}
		}
	}

	close(rf.appStopCh)
}

// consistencyHandler handling the relevant logic in the Raft cluster, such as
// sending heartbeats and processing data that has been applied.
func (rf *raftServer) consistencyHandler() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer func() {
		err := rf.stableStorage.Close()
		if err != nil {
			rf.logger.Error("stable storage close error", zap.Error(err))
		}
		ticker.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			rf.node.Tick()

		case err := <-rf.transport.ErrorC:
			rf.stopError(err)
			return

		case <-rf.appStopCh:
			rf.stopRaft()
			return

		case ready := <-rf.node.Ready():
			// Store raft entries to wal, then publish over commit channel.
			// Must save the snapshot file and WAL snapshot entry before
			// saving any other entries or hard state to ensure that recovery
			// after a snapshot restore is possible.
			if !raft.IsEmptySnap(ready.Snapshot) {
				err := rf.saveSnapshot(ready.Snapshot)
				if err != nil {
					rf.logger.Error("node save snapshot failed", zap.Error(err))
				}
			}
			if err := rf.stableStorage.Save(ready.HardState, ready.Entries); err != nil {
				rf.logger.Panic("node hard state save error",
					zap.Uint64("term", ready.HardState.Term),
					zap.Uint64("commit", ready.Commit),
					zap.Error(err),
				)
			}
			if !raft.IsEmptySnap(ready.Snapshot) {
				err := rf.unstableStorage.ApplySnapshot(ready.Snapshot)
				if err != nil {
					rf.logger.Error("unstable storage apply error", zap.Error(err))
				}
				rf.publishSnapshot(ready.Snapshot)
			}
			if err := rf.unstableStorage.Append(ready.Entries); err != nil {
				rf.logger.Panic("log entries append unstably error", zap.Error(err))
			}
			rf.transport.Send(rf.processMessages(ready.Messages))
			applyDoneCh, ok := rf.publishEntries(rf.needApplyEntries(ready.CommittedEntries))
			if !ok {
				rf.stopRaft()
				return
			}

			// soft state change should update leader
			if ready.SoftState != nil {
				rf.cluster.update(types.ID(ready.SoftState.Lead), raft.StateLeader)
			}
			rf.maybeTriggerSnapshot(applyDoneCh)
			rf.node.Advance()
		}
	}
}

func (rf *raftServer) stopRaft() {
	rf.stopTransport()
	close(rf.commitCh)
	close(rf.errorCh)
	rf.node.Stop()
}

func (rf *raftServer) stopError(err error) {
	rf.stopTransport()
	close(rf.commitCh)
	rf.errorCh <- err
	close(rf.errorCh)
	rf.node.Stop()
}

func (rf *raftServer) stopTransport() {
	rf.transport.Stop()
	close(rf.httpStopCh)
	<-rf.httpDoneCh
}

// When there is a `raftpb.EntryConfChange` after creating the snapshot,
// then the confState included in the snapshot is out of date. so We need
// to update the confState before sending a snapshot to a follower.
func (rf *raftServer) processMessages(msg []raftpb.Message) []raftpb.Message {
	for i := 0; i < len(msg); i++ {
		if msg[i].Type == raftpb.MsgSnap {
			msg[i].Snapshot.Metadata.ConfState = rf.confState
		}
	}
	return msg
}

// needApplyEntries return committed entries which need to be applied by application layer.
func (rf *raftServer) needApplyEntries(entries []raftpb.Entry) []raftpb.Entry {
	if len(entries) == 0 {
		return entries
	}

	firstIndex := entries[0].Index
	if firstIndex > rf.appliedIndex+1 {
		rf.logger.Panic("first index bigger than applied",
			zap.Uint64("first", firstIndex),
			zap.Uint64("applied", rf.appliedIndex),
		)
	}
	result := make([]raftpb.Entry, 0)
	if rf.appliedIndex-firstIndex+1 < uint64(len(entries)) {
		result = entries[rf.appliedIndex-firstIndex+1:]
	}

	return result
}

// publishEntries writes committed log entries to commit channel and returns
// whether all entries could be published.
func (rf *raftServer) publishEntries(entries []raftpb.Entry) (<-chan struct{}, bool) {
	if len(entries) == 0 {
		return nil, true
	}

	data := make([]string, 0, len(entries))
	for i := range entries {
		switch entries[i].Type {
		case raftpb.EntryNormal:
			if len(entries[i].Data) > 0 {
				data = append(data, string(entries[i].Data))
			}
		case raftpb.EntryConfChange:
			var confChange raftpb.ConfChange
			_ = confChange.Unmarshal(entries[i].Data)
			if rf.processConfChange(confChange) == false {
				return nil, false
			}
		}
	}

	// After commit to application layer, update appliedIndex.
	var applyDoneCh chan struct{}
	if len(data) > 0 {
		applyDoneCh = make(chan struct{}, 1)
		select {
		case rf.commitCh <- &Commit{data, applyDoneCh}:
		case <-rf.appStopCh:
			return nil, false
		}
	}

	rf.appliedIndex = entries[len(entries)-1].Index
	return applyDoneCh, true
}

func (rf *raftServer) processConfChange(confChange raftpb.ConfChange) bool {
	rf.confState = *rf.node.ApplyConfChange(confChange)
	switch confChange.Type {
	case raftpb.ConfChangeAddNode:
		if len(confChange.Context) == 0 {
			return true
		}
		rf.cluster.join(types.ID(confChange.NodeID), []string{string(confChange.Context)})
		rf.transport.AddPeer(types.ID(confChange.NodeID), []string{string(confChange.Context)})
	case raftpb.ConfChangeRemoveNode:
		if confChange.NodeID == uint64(rf.cluster.self) {
			rf.logger.Info("remove node from cluster, shutting down")
			return false
		}
		rf.cluster.leave(types.ID(confChange.NodeID))
		rf.transport.RemovePeer(types.ID(confChange.NodeID))
	}
	return true
}

// maybeTriggerSnapshot will judge whether create snapshot or not, when entries
// applied by application layer. There has a compact for snapshot.
func (rf *raftServer) maybeTriggerSnapshot(applyDoneCh <-chan struct{}) {
	if rf.appliedIndex-rf.snapshotIndex <= defaultSnapshotLimit {
		return
	}
	// wait until entries applied (or server shut down)
	if applyDoneCh != nil {
		select {
		case <-applyDoneCh:
		case <-rf.appStopCh:
			return
		}
	}

	rf.logger.Info("snapshot apply log index", zap.Uint64("from", rf.snapshotIndex), zap.Uint64("to", rf.appliedIndex))
	rawData, err := rf.snapshotFetch()
	if err != nil {
		rf.logger.Error("fetch application snapshot error", zap.Error(err))
		return
	}
	snapshot, err := rf.unstableStorage.CreateSnapshot(rf.appliedIndex, &rf.confState, rawData)
	if err != nil {
		rf.logger.Error("create snapshot failed", zap.Error(err))
		return
	}
	if err = rf.saveSnapshot(snapshot); err != nil {
		rf.logger.Error("save snapshot failed", zap.Error(err))
		return
	}

	// compact log entries for snapshot entries
	var compactIndex uint64 = 1
	if rf.appliedIndex > defaultCompactLimit {
		compactIndex = rf.appliedIndex - defaultCompactLimit
	}
	err = rf.unstableStorage.Compact(compactIndex)
	if err != nil {
		rf.logger.Panic("snapshot can not be compacted", zap.Error(err))
	}

	rf.snapshotIndex = rf.appliedIndex
}
