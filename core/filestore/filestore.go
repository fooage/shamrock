package filestore

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/fooage/shamrock/core/raft"
	"github.com/fooage/shamrock/utils"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
	"go.uber.org/zap"
)

type FileStorage interface {
	Lookup(key string) ([]byte, bool)
	Propose(key string, value []byte) error
	SnapshotFetch() ([]byte, error)
	Connect(cluster raft.Cluster)
}

// The file store relies heavily on disk for storage and its role is to store
// chunks of object, with the possibility of adding an in-memory cache in the future.

type filestoreServer struct {
	filestore map[string]*os.File
	snapshot  *snap.Snapshotter
	mutex     sync.RWMutex
	cluster   raft.Cluster

	storePath string // file store path prefix
	logger    *zap.Logger
}

func (f *filestoreServer) Lookup(key string) ([]byte, bool) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	// TODO: Check if there are concurrency issues here.
	file, ok := f.filestore[key]
	if !ok {
		return nil, ok
	}
	data, err := f.bufferReadFile(file)
	if err != nil {
		f.logger.Error("buffer read file failed", zap.Error(err))
		return nil, false
	}
	return data, true
}

// The Chunk structure is for better serialization of data between the file store
// layer and the raft layer. Encode and decode is according to its fields.
type Chunk struct {
	Key   string
	Value []byte
}

func (f *filestoreServer) Propose(key string, value []byte) (err error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	// TODO: Check if there are concurrency issues here.
	file, ok := f.filestore[key]
	if !ok {
		file, err = os.Create(fmt.Sprintf("%s/%s", f.storePath, key))
		if err != nil {
			f.logger.Error("system create file error", zap.Error(err))
			return err
		}
		f.filestore[key] = file
	}
	err = f.bufferWriteFile(file, value)
	if err != nil {
		f.logger.Error("write file failed", zap.String("file", file.Name()), zap.Error(err))
		return err
	}

	// After writing to the file, the binary data of its file is synchronized
	// to other nodes through Raft.
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err = encoder.Encode(Chunk{Key: key, Value: value})
	if err != nil {
		f.logger.Error("can not encode file as key-value", zap.Error(err))
		return err
	}
	f.cluster.Propose() <- buf.String()
	return nil
}

func (f *filestoreServer) SnapshotFetch() ([]byte, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return json.Marshal(f.filestore)
}

func (f *filestoreServer) Connect(cluster raft.Cluster) {
	f.cluster = cluster
	f.snapshot = <-f.cluster.Snapshot()
	snapshot, err := f.loadFromSnapshot()
	if err != nil {
		f.logger.Panic("load from snapshot error", zap.Error(err))
	}
	if snapshot != nil {
		err = f.recoverFromSnapshot(snapshot)
		if err != nil {
			f.logger.Panic("recover KV from snapshot failed", zap.Error(err))
		}
	}

	// Read commits from raft into filestore until error.
	go f.readCommits()
}

func NewFileStoreServer(logger *zap.Logger, storePath string) FileStorage {
	if !utils.PathExist(storePath) {
		if err := os.Mkdir(storePath, 0750); err != nil {
			logger.Panic("create store folder error", zap.String("path", storePath), zap.Error(err))
		}
	}
	return &filestoreServer{
		filestore: make(map[string]*os.File),
		storePath: storePath,
		logger:    logger,
	}
}

// readCommits is a main loop to continuously read committed data from raft layer
// and apply it to the actual file storage until the error happen.
func (f *filestoreServer) readCommits() {
	for {
		select {
		case err := <-f.cluster.Error():
			f.logger.Panic("stop read raft commits", zap.Error(err))

		case commit := <-f.cluster.Commit():
			if commit == nil {
				snapshot, err := f.loadFromSnapshot()
				if err != nil {
					f.logger.Panic("load from snapshot error", zap.Error(err))
				}
				if snapshot != nil {
					err := f.recoverFromSnapshot(snapshot)
					if err != nil {
						f.logger.Panic("recover Map from snapshot failed", zap.Error(err))
					}
				}
				continue
			}

			for _, dataStr := range commit.Data {
				data := Chunk{Key: "", Value: nil}
				decoder := gob.NewDecoder(bytes.NewBufferString(dataStr))
				err := decoder.Decode(&data)
				if err != nil {
					f.logger.Error("can not decode message as key-value", zap.Error(err))
				}
				f.mutex.Lock()
				file, ok := f.filestore[data.Key]
				if !ok {
					file, err = os.Create(fmt.Sprintf("%s/%s", f.storePath, data.Key))
					if err != nil {
						f.logger.Error("synchronized file create error", zap.String("file", data.Key), zap.Error(err))
					}
				}
				if err := f.bufferWriteFile(file, data.Value); err != nil {
					f.logger.Error("data write on disk failed", zap.String("file", file.Name()), zap.Error(err))
					continue
				}
				f.filestore[data.Key] = file
				f.mutex.Unlock()
			}

			// Notify Raft that the message has been applied by closing the channel.
			close(commit.ApplyDoneCh)
		}
	}
}

func (f *filestoreServer) loadFromSnapshot() (*raftpb.Snapshot, error) {
	snapshot, err := f.snapshot.Load()
	if err == snap.ErrNoSnapshot {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return snapshot, nil
}

// Since lightweight snapshots only record a list of files, need to actively
// pull and overwrite files from the Leader node when restoring from snapshots.
func (f *filestoreServer) recoverFromSnapshot(snapshot *raftpb.Snapshot) error {
	f.logger.Info("recover from snapshot meta data",
		zap.Uint64("term", snapshot.Metadata.Term),
		zap.Uint64("index", snapshot.Metadata.Index),
	)
	var filestore map[string]*os.File
	if err := json.Unmarshal(snapshot.Data, &filestore); err != nil {
		return err
	}

	f.mutex.Lock()
	if _, ok := f.cluster.Leader(); ok {
		// TODO: Complete logic which pull chunks from Leader node.
	} else {
		return errors.New("cluster has no leader, can not sync")
	}
	f.filestore = filestore
	f.mutex.Unlock()
	return nil
}
