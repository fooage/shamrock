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

	"github.com/fooage/shamrock/core/raft"
	"github.com/fooage/shamrock/proto/proto_gen/block_service"
	"github.com/fooage/shamrock/utils"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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
	storePath string

	logger *zap.Logger
}

func NewFileStoreServer(logger *zap.Logger) FileStorage {
	return &filestoreServer{
		filestore: make(map[string]*os.File),
		storePath: "",
		logger:    logger,
	}
}

func (f *filestoreServer) Lookup(key string) ([]byte, bool) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()

	// TODO: Check if there are concurrency issues here.
	file, ok := f.filestore[key]
	if !ok {
		return nil, ok
	}
	file, err := os.OpenFile(file.Name(), os.O_RDONLY, os.ModePerm)
	if err != nil {
		f.logger.Error("open file error", zap.Error(err))
		return nil, false
	}
	defer file.Close()
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
	Hash   string
	Binary []byte
}

func (f *filestoreServer) Propose(key string, value []byte) (err error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err = encoder.Encode(Chunk{Hash: key, Binary: value})
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
	f.storePath = fmt.Sprintf("store-%s-%s", cluster.Group(), cluster.Self())
	if !utils.PathExist(f.storePath) {
		if err := os.Mkdir(f.storePath, 0750); err != nil {
			f.logger.Panic("create store folder error", zap.String("path", f.storePath), zap.Error(err))
		}
	}

	// Restore data from the Raft snapshot of the consistency layer.
	f.cluster = cluster
	f.snapshot = <-f.cluster.Snapshot()
	snapshot, err := f.loadFromSnapshot()
	if err != nil {
		f.logger.Panic("load from snapshot error", zap.Error(err))
	}
	if snapshot != nil {
		err = f.recoverFromSnapshot(snapshot)
		if err != nil {
			f.logger.Panic("recover Map from snapshot failed", zap.Error(err))
		}
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
				data := Chunk{Hash: "", Binary: nil}
				decoder := gob.NewDecoder(bytes.NewBufferString(dataStr))
				err := decoder.Decode(&data)
				if err != nil {
					f.logger.Error("can not decode message as key-value", zap.Error(err))
					continue
				}
				f.mutex.Lock()
				file, ok := f.filestore[data.Hash]
				if !ok {
					file, err = os.OpenFile(fmt.Sprintf("%s/%s", f.storePath, data.Hash), os.O_CREATE|os.O_TRUNC, os.ModePerm)
					if err != nil {
						f.logger.Error("synchronized file create error", zap.String("file", data.Hash), zap.Error(err))
					}
				}
				if err := f.bufferWriteFile(file, data.Binary); err != nil {
					f.logger.Error("data write on disk failed", zap.String("file", file.Name()), zap.Error(err))
					continue
				}
				f.filestore[data.Hash] = file
				file.Close()
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
	for hash, _ := range filestore {
		chunkResp, err := client.GetChunk(context.TODO(), &block_service.GetChunkReq{
			Hash:       hash,
			FromMaster: true,
		})
		if err != nil {
			f.logger.Error("get chunk from master error", zap.Error(err), zap.String("hash", hash))
			return err
		}
		filePath := fmt.Sprintf("%s/%s", f.storePath, chunkResp.Hash)
		if !utils.PathExist(filePath) {
			file, err := os.OpenFile(filePath, os.O_CREATE|os.O_TRUNC, os.ModePerm)
			if err != nil {
				f.logger.Error("synchronized file create error", zap.String("file", chunkResp.Hash), zap.Error(err))
			}
			if err := f.bufferWriteFile(file, chunkResp.Binary); err != nil {
				f.logger.Error("data write on disk failed", zap.String("file", file.Name()), zap.Error(err))
				return err
			}
			f.mutex.Lock()
			f.filestore[chunkResp.Hash] = file
			f.mutex.Unlock()
		}
	}
	return nil
}
