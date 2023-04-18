package kvstore

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"sync"

	"github.com/fooage/shamrock/core/raft"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
	"go.uber.org/zap"
)

type KVStorage interface {
	Lookup(key string) (string, bool)
	Propose(key string, value string) error
	SnapshotFetch() ([]byte, error)
	Connect(cluster raft.Cluster)
}

// A very simple KV storage which is based on Raft and is used to store the
// file routing table on meta server, also store the file table in each filestore.

type kvstoreServer struct {
	kvstore  map[string]string
	snapshot *snap.Snapshotter
	mutex    sync.RWMutex
	cluster  raft.Cluster

	logger *zap.Logger
}

func NewKVStoreServer(logger *zap.Logger) KVStorage {
	return &kvstoreServer{
		kvstore: make(map[string]string),
		logger:  logger,
	}
}

// The KV structure is for better serialization of data between the key-value
// layer and the raft layer. Encode and decode is according to its fields.
type KV struct {
	Key   string
	Value string
}

func (kv *kvstoreServer) Lookup(key string) (string, bool) {
	kv.mutex.RLock()
	value, ok := kv.kvstore[key]
	kv.mutex.RUnlock()
	return value, ok
}

func (kv *kvstoreServer) Propose(key string, value string) error {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(KV{Key: key, Value: value})
	if err != nil {
		kv.logger.Error("can not encode message as key-value", zap.Error(err))
		return err
	}
	kv.cluster.Propose() <- buf.String()
	return nil
}

func (kv *kvstoreServer) SnapshotFetch() ([]byte, error) {
	kv.mutex.RLock()
	defer kv.mutex.RUnlock()
	return json.Marshal(kv.kvstore)
}

func (kv *kvstoreServer) Connect(cluster raft.Cluster) {
	kv.cluster = cluster
	kv.snapshot = <-kv.cluster.Snapshot()
	snapshot, err := kv.loadFromSnapshot()
	if err != nil {
		kv.logger.Panic("load from snapshot error", zap.Error(err))
	}
	if snapshot != nil {
		err = kv.recoverFromSnapshot(snapshot)
		if err != nil {
			kv.logger.Panic("recover KV from snapshot failed", zap.Error(err))
		}
	}

	// Read commits from raft into kvstore until error.
	go kv.readCommits(kv.cluster.Commit(), kv.cluster.Error())
}

// readCommits is a main loop to continuously read committed data from raft layer
// and apply it to the actual key-value storage until the error happen.
func (kv *kvstoreServer) readCommits(commitCh <-chan *raft.Commit, errorCh <-chan error) {
	for {
		select {
		case err := <-errorCh:
			kv.logger.Panic("stop read raft commits", zap.Error(err))

		case commit := <-commitCh:
			if commit == nil {
				snapshot, err := kv.loadFromSnapshot()
				if err != nil {
					kv.logger.Panic("load from snapshot error", zap.Error(err))
				}
				if snapshot != nil {
					err := kv.recoverFromSnapshot(snapshot)
					if err != nil {
						kv.logger.Panic("recover KV from snapshot failed", zap.Error(err))
					}
				}
				continue
			}

			for _, dataStr := range commit.Data {
				data := KV{Key: "", Value: ""}
				decoder := gob.NewDecoder(bytes.NewBufferString(dataStr))
				err := decoder.Decode(&data)
				if err != nil {
					kv.logger.Error("can not decode message as key-value", zap.Error(err), zap.String("data", dataStr))
				}
				kv.mutex.Lock()
				kv.kvstore[data.Key] = data.Value
				kv.mutex.Unlock()
			}

			// Notify Raft that the message has been applied by closing the channel.
			close(commit.ApplyDoneCh)
		}
	}
}

func (kv *kvstoreServer) loadFromSnapshot() (*raftpb.Snapshot, error) {
	snapshot, err := kv.snapshot.Load()
	if err == snap.ErrNoSnapshot {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (kv *kvstoreServer) recoverFromSnapshot(snapshot *raftpb.Snapshot) error {
	kv.logger.Info("recover from snapshot meta data",
		zap.Uint64("term", snapshot.Metadata.Term),
		zap.Uint64("index", snapshot.Metadata.Index),
	)
	var kvstore map[string]string
	if err := json.Unmarshal(snapshot.Data, &kvstore); err != nil {
		return err
	}
	kv.mutex.Lock()
	kv.kvstore = kvstore
	kv.mutex.Unlock()
	return nil
}
