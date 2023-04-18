package raft

import (
	"go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
	"go.etcd.io/etcd/server/v3/wal"
	"go.etcd.io/etcd/server/v3/wal/walpb"
	"go.uber.org/zap"
)

// loadSnapshot load the latest log stored in WAL.
func (rf *raftServer) loadSnapshot() *raftpb.Snapshot {
	if wal.Exist(rf.walPath) {
		rawData, err := wal.ValidSnapshotEntries(rf.logger, rf.walPath)
		if err != nil {
			rf.logger.Panic("loading snapshot in WAL error", zap.Error(err))
		}
		snapshot, err := rf.snapshot.LoadNewestAvailable(rawData)
		if err != nil && err != snap.ErrNoSnapshot {
			rf.logger.Panic("read raw snapshot error", zap.Error(err))
		}
		return snapshot
	}
	return &raftpb.Snapshot{}
}

// saveSnapshot save snapshot first, WAL second.
func (rf *raftServer) saveSnapshot(snap raftpb.Snapshot) error {
	rawData := walpb.Snapshot{
		Index:     snap.Metadata.Index,
		Term:      snap.Metadata.Term,
		ConfState: &snap.Metadata.ConfState,
	}

	// Save the snapshot file before writing the snapshot to the WAL.
	// This makes it possible for the snapshot file to become orphaned, but prevents
	// a WAL snapshot entry from having no corresponding snapshot file.
	err := rf.snapshot.SaveSnap(snap)
	if err != nil {
		rf.logger.Error("snapshot save failed", zap.Error(err))
		return err
	}
	err = rf.stableStorage.SaveSnapshot(rawData)
	if err != nil {
		rf.logger.Error("snapshot save stably failed", zap.Error(err))
		return err
	}

	return rf.stableStorage.ReleaseLockTo(snap.Metadata.Index)
}

// publishSnapshot resolve committed and applied snapshot, and change log index.
func (rf *raftServer) publishSnapshot(snap raftpb.Snapshot) {
	if raft.IsEmptySnap(snap) {
		return
	}
	if snap.Metadata.Index <= rf.appliedIndex {
		rf.logger.Panic("snapshot index smaller than applied",
			zap.Uint64("snapshot", snap.Metadata.Index),
			zap.Uint64("applied", rf.appliedIndex),
		)
	}
	rf.commitCh <- nil
	rf.confState = snap.Metadata.ConfState
	rf.snapshotIndex = snap.Metadata.Index
	rf.appliedIndex = snap.Metadata.Index
}
