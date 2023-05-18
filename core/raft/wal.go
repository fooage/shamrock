package raft

import (
	"os"

	"go.etcd.io/etcd/raft/v3"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.etcd.io/etcd/server/v3/wal"
	"go.etcd.io/etcd/server/v3/wal/walpb"
	"go.uber.org/zap"
)

// openWAL returns a WAL ready for reading.
func (rf *raftServer) openWAL(snapshot *raftpb.Snapshot) *wal.WAL {
	if !wal.Exist(rf.walPath) {
		if err := os.MkdirAll(rf.walPath, 0750); err != nil {
			rf.logger.Panic("make WAL folder error", zap.String("path", rf.walPath), zap.Error(err))
		}
		// create a new ins first
		ins, err := wal.Create(zap.NewExample(), rf.walPath, nil)
		if err != nil {
			rf.logger.Panic("create WAL wal instance error", zap.String("path", rf.walPath), zap.Error(err))
		}
		if err = ins.Close(); err != nil {
			rf.logger.Panic("close new WAL instance error", zap.Error(err))
		}
	}

	// If there exist WAL, then record according to the latest snapshot.
	rawData := walpb.Snapshot{}
	if snapshot != nil {
		rawData.Index = snapshot.Metadata.Index
		rawData.Term = snapshot.Metadata.Term
	}
	ins, err := wal.Open(zap.NewExample(), rf.walPath, rawData)
	if err != nil {
		rf.logger.Panic("open existing WAL instance error", zap.String("path", rf.walPath), zap.Error(err))
	}

	return ins
}

// replayWAL replays WAL entries into the raft instance. This allows the
// previous log to be applied to the current raft state machine.
func (rf *raftServer) replayWAL() *wal.WAL {
	snapshot := rf.loadSnapshot()
	ins := rf.openWAL(snapshot)
	_, state, entries, err := ins.ReadAll()
	if err != nil {
		rf.logger.Panic("WAL read all snapshot error", zap.Error(err))
	}

	// Recover the in-memory storage from persistent snapshot, state and entries.
	rf.unstableStorage = raft.NewMemoryStorage()
	if snapshot != nil {
		err = rf.unstableStorage.ApplySnapshot(*snapshot)
	}
	err = rf.unstableStorage.SetHardState(state)
	err = rf.unstableStorage.Append(entries)
	if err != nil {
		rf.logger.Panic("unstably load snapshot error", zap.Error(err))
	}

	return ins
}
