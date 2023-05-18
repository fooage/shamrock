package filestore

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/fooage/shamrock/core/raft"
	"github.com/fooage/shamrock/proto/proto_gen/block_service"
	"github.com/fooage/shamrock/utils"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type FileStorage interface {
	Lookup(key string) (*block_service.BlockEntry, bool)
	Propose(command CommandType, entry *block_service.BlockEntry) error
	SnapshotFetch() ([]byte, error)
	Connect(cluster raft.Cluster)
}

// The file store relies heavily on disk for storage and its role is to store
// chunks of object, with the possibility of adding an in-memory cache in the future.

type filestoreServer struct {
	filestore   map[string]*block_service.BlockEntry
	proposeDone sync.Map
	snapshot    *snap.Snapshotter
	mutex       sync.RWMutex
	cluster     raft.Cluster
	storePath   string
	logger      *zap.Logger
}

func NewFileStoreServer(logger *zap.Logger) FileStorage {
	return &filestoreServer{
		filestore: make(map[string]*block_service.BlockEntry),
		storePath: "",
		logger:    logger,
	}
}

func (f *filestoreServer) Lookup(key string) (*block_service.BlockEntry, bool) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	storeInfo, ok := f.filestore[key]
	if !ok {
		f.logger.Warn("file not found", zap.String("file", key))
		return nil, ok
	}
	file, err := os.OpenFile(f.GenerateFilePath(storeInfo.UniqueKey), os.O_RDONLY, os.ModePerm)
	if err != nil {
		f.logger.Error("open file error", zap.Error(err))
		return nil, false
	}
	defer file.Close()
	data, err := utils.BufferReadFile(file)
	if err != nil {
		f.logger.Error("buffer read file failed", zap.Error(err))
		return nil, false
	}
	return &block_service.BlockEntry{
		UniqueKey: storeInfo.UniqueKey,
		Hash:      storeInfo.Hash,
		Binary:    data,
	}, true
}

// The LogEntry structure is for better serialization of data between the file store
// layer and the raft layer. Encode and decode is according to its fields.
type LogEntry struct {
	Command CommandType
	Data    *block_service.BlockEntry
}

type CommandType string

const (
	StoreCommand  CommandType = "STORE"
	DeleteCommand CommandType = "DELETE"
)

func (f *filestoreServer) Propose(command CommandType, entry *block_service.BlockEntry) (err error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err = encoder.Encode(LogEntry{Command: command, Data: entry})
	if err != nil {
		f.logger.Error("can not encode file as key-value", zap.Error(err))
		return err
	}

	f.proposeDone.Store(entry.UniqueKey, make(chan struct{}))
	f.cluster.Propose() <- buf.String()
	defer f.proposeDone.Delete(entry.UniqueKey)
	if proposeDone, ok := f.proposeDone.Load(entry.UniqueKey); ok {
		select {
		case <-proposeDone.(chan struct{}):
			f.logger.Info("file data propose done", zap.String("key", entry.UniqueKey))
			return nil
		case <-time.After(5 * time.Second):
			return errors.New("propose wait time out")
		}
	}
	return errors.New("done channel load error")
}

func (f *filestoreServer) SnapshotFetch() ([]byte, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	return json.Marshal(f.filestore)
}

func (f *filestoreServer) Connect(cluster raft.Cluster) {
	f.storePath = fmt.Sprintf("./storage/file-%s-%s", cluster.Group(), cluster.Self())
	if !utils.PathExist(f.storePath) {
		if err := os.MkdirAll(f.storePath, 0750); err != nil {
			f.logger.Panic("create store folder error", zap.String("path", f.storePath), zap.Error(err))
		}
	}

	// Restore data from the Raft snapshot of the consistency layer.
	f.cluster = cluster
	f.snapshot = <-f.cluster.Snapshot()
	err := f.recoverFromSnapshot()
	if err != nil {
		f.logger.Panic("recover Map from snapshot failed", zap.Error(err))
	}

	// Read commits from raft into filestore until error.
	go f.readCommits()
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
				err := f.recoverFromSnapshot()
				if err != nil {
					f.logger.Panic("recover Map from snapshot failed", zap.Error(err))
				}
			} else {
				for _, dataStr := range commit.Data {
					err := f.processCommitData(&dataStr)
					if err != nil {
						f.logger.Warn("process commit data failed", zap.Error(err))
						continue
					}
				}
			}

			// Notify Raft that the message has been applied by closing the channel.
			close(commit.ApplyDoneCh)
		}
	}
}

func (f *filestoreServer) processCommitData(dataStr *string) error {
	entry := LogEntry{}
	decoder := gob.NewDecoder(bytes.NewBufferString(*dataStr))
	err := decoder.Decode(&entry)
	if err != nil {
		f.logger.Error("can not decode message as key-value", zap.Error(err))
		return err
	}

	switch entry.Command {
	case StoreCommand:
		err := f.writeBlockLocked(entry.Data)
		if err != nil {
			f.logger.Error("write file with commit error", zap.Error(err))
			return err
		}
	case DeleteCommand:
		err := f.removeBlockLocked(entry.Data)
		if err != nil {
			f.logger.Error("remove file with commit error", zap.Error(err))
			return err
		}
	default:
		return errors.New("entry command incorrect")
	}

	// Here it is necessary to notify the upper layer that Raft has completed
	// the synchronization of entry, and it can be considered that the entry is
	// written successfully.
	if proposeDone, ok := f.proposeDone.Load(entry.Data.UniqueKey); ok {
		proposeDone.(chan struct{}) <- struct{}{}
	}
	return nil
}

// Since lightweight snapshots only record a list of files, need to actively
// pull and overwrite files from the Leader node when restoring from snapshots.
func (f *filestoreServer) recoverFromSnapshot() error {
	snapshot, err := f.snapshot.Load()
	if err == snap.ErrNoSnapshot {
		return nil
	} else if err != nil {
		f.logger.Error("load from snapshot error", zap.Error(err))
		return err
	}

	f.logger.Info("recover from snapshot meta data",
		zap.Uint64("term", snapshot.Metadata.Term),
		zap.Uint64("index", snapshot.Metadata.Index),
	)
	var filestore map[string]*os.File
	if err := json.Unmarshal(snapshot.Data, &filestore); err != nil {
		f.logger.Panic("snapshot data can not be unmarshal", zap.Error(err))
	}

	// Since the snapshot only has a list of chunks, the data must be pulled
	// from the Leader node.
	if leader, ok := f.cluster.Leader(); ok {
		target := leader[rand.Intn(len(leader))]
		err := f.synchronizedLeader(target, filestore)
		if err != nil {
			f.logger.Error("sync from leader failed", zap.Error(err))
			return err
		}
		return nil
	} else {
		return errors.New("cluster has no leader, can not sync")
	}
}

func (f *filestoreServer) synchronizedLeader(leader url.URL, filestore map[string]*os.File) error {
	connect, err := grpc.Dial(utils.AddressOffsetRPC(leader))
	if err != nil {
		f.logger.Error("grpc dial to master node error", zap.Error(err))
		return err
	}
	defer connect.Close()
	client := block_service.NewBlockServiceClient(connect)

	// Pull files from the Leader node and store them, and all files are synchronized to be successful.
	for uniqueKey := range filestore {
		blockResp, err := client.FetchBlock(context.TODO(), &block_service.FetchBlockReq{
			UniqueKey:  uniqueKey,
			FromMaster: true,
		})
		if err != nil {
			f.logger.Error("get chunk from master error", zap.Error(err), zap.String("uniqueKey", uniqueKey))
			return err
		}
		err = f.writeBlockLocked(blockResp.Entry)
		if err != nil {
			f.logger.Error("synchronized from leader write file error", zap.Error(err))
			return err
		}
	}
	return nil
}
