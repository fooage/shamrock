// Package is a simple Shamrock system SDK used by the client, which has a
// series of atomic operations on storage objects to meet the needs of users.

package client

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/url"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/fooage/shamrock/core/filestore"
	"github.com/fooage/shamrock/core/kvstore"
	"github.com/fooage/shamrock/proto/proto_gen/block_service"
	"github.com/fooage/shamrock/proto/proto_gen/meta_service"
	"github.com/fooage/shamrock/service"
	"github.com/fooage/shamrock/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type Client interface {
	// Fetch function pull storage object based on unique key and return local storage path.
	Fetch(key string) (string, error)
	// Register local file to shamrock system, prepare for store it.
	Register(path string) (string, error)
	// Store will upload the corresponding unique key file after registration.
	Store(key string, path string) error
	// Delete the chunks stored on the block service and mark the meta information as deleted.
	Delete(key string) error
	// Query is used as a function to query meta based on name prefix. It's a page load function.
	Query(prefix string, page, pageSize int64) ([]string, int64, error)
}

type DefaultClient struct {
	storePath string
	mutex     sync.RWMutex
	logger    *zap.Logger
}

func NewDefaultClient(logger *zap.Logger, discovery []string) Client {
	rand.Seed(time.Now().UnixNano())
	service.InitServiceDiscovery(logger, "consul", discovery)
	return &DefaultClient{
		storePath: "./storage",
		logger:    logger,
	}
}

func (c *DefaultClient) generateWholePath(uniqueKey string) string {
	return fmt.Sprintf("%s/%s", c.storePath, uniqueKey)
}

func (c *DefaultClient) generateObjectPath(uniqueKey string) string {
	return fmt.Sprintf("%s/%s/%s", c.storePath, uniqueKey, kvstore.ParseObjectName(uniqueKey))
}

func (c *DefaultClient) generateChunkPath(uniqueKey string, parentKey string) string {
	return fmt.Sprintf("%s/%s/%s", c.storePath, parentKey, kvstore.ParseChunkHash(uniqueKey))
}

func (c *DefaultClient) Fetch(key string) (string, error) {
	// If you have downloaded the required files, open directly and return its name.
	expectPath := c.generateObjectPath(key)
	if ok := utils.PathExist(expectPath); ok {
		temp, err := os.OpenFile(expectPath, os.O_RDONLY, os.ModePerm)
		if err != nil {
			c.logger.Error("exist local file open failed", zap.Error(err), zap.String("path", expectPath))
			return "", err
		}
		defer temp.Close()
		info, _ := temp.Stat()
		return info.Name(), nil
	}
	err := os.MkdirAll(c.generateWholePath(key), 0755)
	if err != nil {
		c.logger.Error("create local path failed", zap.Error(err), zap.String("key", key))
		return "", err
	}

	instances := service.DiscoveryClient.Instances("shamrock-meta")
	if len(instances) == 0 {
		err := errors.New("service not found")
		c.logger.Error("meta service not available", zap.Error(err))
		return "", err
	}
	target := instances[rand.Intn(len(instances))]
	connect, err := grpc.Dial(utils.AddressOffsetRPC(target), grpc.WithInsecure())
	if err != nil {
		c.logger.Error("grpc dial to meta node error", zap.Error(err))
		return "", err
	}
	defer connect.Close()
	metaClient := meta_service.NewMetaServiceClient(connect)
	objectResp, err := metaClient.QueryObjectMeta(context.TODO(), &meta_service.QueryObjectMetaReq{
		UniqueKey: key,
	})
	if err != nil {
		c.logger.Error("meta service query object failed", zap.Error(err))
		return "", err
	}
	objectMeta := objectResp.Meta
	if objectMeta.Status != meta_service.EntryStatus_Available {
		// can only fetch files that are already fully stored
		err := errors.New("entry not available")
		c.logger.Error("object status not available", zap.Error(err))
		return "", err
	}

	var (
		limit       = 100
		begin       = 0
		end         = int(math.Min(float64(begin+limit), float64(len(objectMeta.ChunkList))))
		chunkMetas  = make([]*meta_service.ChunkMeta, 0, len(objectMeta.ChunkList))
		storeGroups = make(map[int64]struct{}, 0)
	)
	for {
		chunkResp, err := metaClient.QueryChunkMeta(context.TODO(), &meta_service.QueryChunkMetaReq{
			UniqueKeys: objectMeta.ChunkList[begin:end],
		})
		if err != nil || len(chunkResp.Result) != end-begin {
			c.logger.Error("meta service query chunk failed", zap.Error(err))
			return "", err
		}
		for _, chunkMeta := range chunkResp.Result {
			chunkMetas = append(chunkMetas, chunkMeta)
			storeGroups[chunkMeta.StoreGroup] = struct{}{}
		}
		if len(objectMeta.ChunkList) == end {
			break
		}
		begin = begin + limit
		end = int(math.Min(float64(end+limit), float64(len(objectMeta.ChunkList))))
	}

	storeAddress := make(map[int64][]string, 0)
	for storeGroup := range storeGroups {
		addressResp, err := metaClient.QueryStorageAddress(context.TODO(), &meta_service.QueryStorageAddressReq{
			StoreGroup: storeGroup,
			FromMaster: false,
		})
		if err != nil {
			c.logger.Error("query chunk storage address failed", zap.Error(err))
			return "", err
		}
		storeAddress[storeGroup] = addressResp.AddressList
	}

	var waitGroup sync.WaitGroup
	for _, chunkMeta := range chunkMetas {
		if ok := utils.PathExist(c.generateChunkPath(chunkMeta.UniqueKey, chunkMeta.Parent)); ok {
			continue
		}
		waitGroup.Add(1)
		go func(addressList []string, meta *meta_service.ChunkMeta) {
			defer waitGroup.Done()
			err := c.fetchChunkBinaryData(addressList, meta)
			if err != nil {
				c.logger.Error("concurrent fetching chunk failed", zap.Error(err))
				return
			}
			c.logger.Info("chunk has been stored", zap.String("key", meta.UniqueKey))
		}(storeAddress[chunkMeta.StoreGroup], chunkMeta)
	}
	waitGroup.Wait()
	err = c.stitchCompleteObject(key, chunkMetas)
	if err != nil {
		c.logger.Error("stitch chunks to object error", zap.Error(err))
		return "", err
	}

	return expectPath, nil
}

// fetchChunkBinaryData fetch the specified chunk data to the block storage
// service according to the routing policy and unique key. (Concurrency Safe)
func (c *DefaultClient) fetchChunkBinaryData(addressList []string, meta *meta_service.ChunkMeta) error {
	address, _ := url.Parse(addressList[rand.Intn(len(addressList))])
	connect, err := grpc.Dial(utils.AddressOffsetRPC(*address), grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(2*filestore.DefaultBlockSize)))
	if err != nil {
		c.logger.Error("grpc dial to block node error", zap.Error(err))
		return err
	}
	defer connect.Close()
	blockClient := block_service.NewBlockServiceClient(connect)
	blockResp, err := blockClient.FetchBlock(context.TODO(), &block_service.FetchBlockReq{
		UniqueKey:  meta.UniqueKey,
		FromMaster: false,
	})
	if err != nil {
		c.logger.Error("fetch chunk data failed", zap.Error(err), zap.String("key", meta.UniqueKey), zap.String("target", address.String()))
		return err
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	file, err := os.OpenFile(c.generateChunkPath(meta.UniqueKey, meta.Parent), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		c.logger.Error("file create error", zap.String("file", meta.UniqueKey), zap.Error(err))
		return err
	}
	defer file.Close()
	if err = utils.BufferWriteFile(file, blockResp.Entry.Binary); err != nil {
		c.logger.Error("data write on disk failed", zap.String("file", file.Name()), zap.Error(err))
		return err
	}
	return nil
}

// stitchCompleteObject concat all the chunks belonging to the object
// sequentially and deletes them. (Concurrency Safe)
func (c *DefaultClient) stitchCompleteObject(uniqueKey string, chunkMetas []*meta_service.ChunkMeta) error {
	sort.Slice(chunkMetas, func(i, j int) bool {
		return chunkMetas[i].Index < chunkMetas[j].Index
	})
	c.mutex.Lock()
	defer c.mutex.Unlock()
	file, err := os.OpenFile(c.generateObjectPath(uniqueKey), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		c.logger.Error("create file object file error", zap.Error(err))
		return err
	}
	defer file.Close()
	for _, meta := range chunkMetas {
		chunk, err := os.OpenFile(c.generateChunkPath(meta.UniqueKey, uniqueKey), os.O_RDONLY, os.ModePerm)
		if err != nil {
			c.logger.Error("create file chunk file error", zap.Error(err))
			return err
		}
		chunkData, err := utils.BufferReadFile(chunk)
		if err != nil {
			c.logger.Error("read chunk file data error", zap.Error(err), zap.Int64("index", meta.Index))
			chunk.Close()
			return err
		}
		if err := utils.BufferWriteFile(file, chunkData); err != nil {
			c.logger.Error("merge chunk to file error", zap.Error(err), zap.Int64("index", meta.Index))
			chunk.Close()
			return err
		}
		chunk.Close()
	}
	for _, meta := range chunkMetas {
		err := os.Remove(c.generateChunkPath(meta.UniqueKey, meta.Parent))
		if err != nil {
			c.logger.Error("remove naive chunk failed", zap.Error(err), zap.String("path", c.generateChunkPath(meta.UniqueKey, meta.Parent)))
		}
	}
	return nil
}

func (c *DefaultClient) Register(path string) (string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		c.logger.Error("open file error", zap.Error(err), zap.String("path", path))
		return "", err
	}
	defer file.Close()
	info, _ := file.Stat()

	chunkData := make([]byte, filestore.DefaultBlockSize)
	chunkCount := int(info.Size()/filestore.DefaultBlockSize) + 1
	registerReq := &meta_service.RegisterObjectReq{
		Name:     info.Name(),
		Size:     info.Size(),
		HashList: make([]string, 0, chunkCount),
	}
	for index := 0; index < chunkCount; index++ {
		size, err := file.Read(chunkData)
		if err != nil && err != io.EOF {
			c.logger.Error("read chunk data error", zap.Error(err))
			return "", err
		}
		hash := md5.New()
		hash.Write(chunkData[:size])
		registerReq.HashList = append(registerReq.HashList, hex.EncodeToString(hash.Sum(nil)))
		if err == io.EOF {
			break
		}
	}

	instances := service.DiscoveryClient.Instances("shamrock-meta")
	if len(instances) == 0 {
		err := errors.New("service not found")
		c.logger.Error("meta service not found", zap.Error(err))
		return "", err
	}
	target := instances[rand.Intn(len(instances))]
	connect, err := grpc.Dial(utils.AddressOffsetRPC(target), grpc.WithInsecure())
	if err != nil {
		c.logger.Error("grpc dial to meta node error", zap.Error(err))
		return "", err
	}
	defer connect.Close()
	metaClient := meta_service.NewMetaServiceClient(connect)
	registerResp, err := metaClient.RegisterObject(context.TODO(), registerReq)
	if err != nil {
		c.logger.Error("register store object failed", zap.Error(err))
		return "", err
	}

	return registerResp.Meta.UniqueKey, nil
}

func (c *DefaultClient) Store(uniqueKey string, path string) error {
	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		c.logger.Error("open file error", zap.Error(err), zap.String("path", path))
		return err
	}
	defer file.Close()
	instances := service.DiscoveryClient.Instances("shamrock-meta")
	if len(instances) == 0 {
		err := errors.New("service not found")
		c.logger.Error("meta service not found", zap.Error(err))
		return err
	}
	target := instances[rand.Intn(len(instances))]
	connect, err := grpc.Dial(utils.AddressOffsetRPC(target), grpc.WithInsecure())
	if err != nil {
		c.logger.Error("grpc dial to meta node error", zap.Error(err))
		return err
	}
	defer connect.Close()
	metaClient := meta_service.NewMetaServiceClient(connect)
	objectResp, err := metaClient.QueryObjectMeta(context.TODO(), &meta_service.QueryObjectMetaReq{
		UniqueKey: uniqueKey,
	})
	if err != nil {
		c.logger.Error("meta service query object failed", zap.Error(err))
		return err
	}
	objectMeta := objectResp.Meta
	if objectMeta == nil || objectMeta.Status == meta_service.EntryStatus_Deleted {
		c.logger.Info("object status not allow register", zap.String("key", uniqueKey))
		return nil
	}
	_, err = metaClient.UpdateObjectStatus(context.TODO(), &meta_service.UpdateObjectStatusReq{
		UniqueKey: uniqueKey,
		Status:    meta_service.EntryStatus_Storing,
	})
	if err != nil {
		c.logger.Error("object meta status update error", zap.Error(err))
		return err
	}

	var (
		limit       = 100
		begin       = 0
		end         = int(math.Min(float64(begin+limit), float64(len(objectMeta.ChunkList))))
		chunkMetas  = make([]*meta_service.ChunkMeta, 0, len(objectMeta.ChunkList))
		storeGroups = make(map[int64]struct{}, 0)
	)
	for {
		chunkResp, err := metaClient.QueryChunkMeta(context.TODO(), &meta_service.QueryChunkMetaReq{
			UniqueKeys: objectMeta.ChunkList[begin:end],
		})
		if err != nil || len(chunkResp.Result) != end-begin {
			c.logger.Error("meta service query chunk failed", zap.Error(err))
			return err
		}
		for _, chunkMeta := range chunkResp.Result {
			chunkMetas = append(chunkMetas, chunkMeta)
			storeGroups[chunkMeta.StoreGroup] = struct{}{}
		}
		if len(objectMeta.ChunkList) == end {
			break
		}
		begin = begin + limit
		end = int(math.Min(float64(end+limit), float64(len(objectMeta.ChunkList))))
	}

	storeAddress := make(map[int64][]string, 0)
	for storeGroup := range storeGroups {
		addressResp, err := metaClient.QueryStorageAddress(context.TODO(), &meta_service.QueryStorageAddressReq{
			StoreGroup: storeGroup,
			FromMaster: false,
		})
		if err != nil {
			c.logger.Error("query chunk storage address failed", zap.Error(err))
			return err
		}
		storeAddress[storeGroup] = addressResp.AddressList
	}

	var waitGroup sync.WaitGroup
	for _, chunkMeta := range chunkMetas {
		if chunkMeta.Status != meta_service.EntryStatus_Storing {
			waitGroup.Add(1)
			go func(addressList []string, meta *meta_service.ChunkMeta) {
				defer waitGroup.Done()
				err := c.storeChunkBinaryData(addressList, meta, file)
				if err != nil {
					c.logger.Error("concurrent fetching chunk failed", zap.Error(err))
					return
				}
				c.logger.Info("chunk has been stored", zap.String("key", meta.UniqueKey))
				_, err = metaClient.UpdateChunkStatus(context.TODO(), &meta_service.UpdateChunkStatusReq{
					UniqueKeys: []string{meta.UniqueKey},
					Status:     meta_service.EntryStatus_Storing,
				})
				if err != nil {
					c.logger.Error("chunk meta status update error", zap.Error(err))
					return
				}
			}(storeAddress[chunkMeta.StoreGroup], chunkMeta)
		}
	}
	waitGroup.Wait()

	_, err = metaClient.UpdateObjectStatus(context.TODO(), &meta_service.UpdateObjectStatusReq{
		UniqueKey: uniqueKey,
		Status:    meta_service.EntryStatus_Available,
	})
	if err != nil {
		c.logger.Error("object meta status update error", zap.Error(err))
		return err
	}
	return nil
}

// storeChunkBinaryData uploads the chunk file fragment data to the specified
// block storage service. (Concurrency Safe)
func (c *DefaultClient) storeChunkBinaryData(addressList []string, meta *meta_service.ChunkMeta, file *os.File) error {
	address, _ := url.Parse(addressList[rand.Intn(len(addressList))])
	connect, err := grpc.Dial(utils.AddressOffsetRPC(*address), grpc.WithInsecure())
	if err != nil {
		c.logger.Error("grpc dial to block node error", zap.Error(err))
		return err
	}
	defer connect.Close()
	c.mutex.Lock()
	blockClient := block_service.NewBlockServiceClient(connect)
	chunkData := make([]byte, filestore.DefaultBlockSize)
	_, err = file.Seek(meta.Index*filestore.DefaultBlockSize, 0)
	if err != nil {
		c.logger.Error("read offset file error", zap.Error(err))
		c.mutex.Unlock()
		return err
	}
	size, err := file.Read(chunkData)
	if err != nil && err != io.EOF {
		c.logger.Error("read chunk data error", zap.Error(err))
		c.mutex.Unlock()
		return err
	}
	c.mutex.Unlock()
	hash := md5.New()
	hash.Write(chunkData[:size])
	_, err = blockClient.StoreBlock(context.TODO(), &block_service.StoreBlockReq{
		Entry: &block_service.BlockEntry{
			UniqueKey: meta.UniqueKey,
			Hash:      hex.EncodeToString(hash.Sum(nil)),
			Binary:    chunkData[:size],
		},
	})
	if err != nil {
		c.logger.Error("block service store chunk failed", zap.Error(err), zap.String("key", meta.UniqueKey))
		return err
	}
	return nil
}

func (c *DefaultClient) Delete(uniqueKey string) error {
	instances := service.DiscoveryClient.Instances("shamrock-meta")
	if len(instances) == 0 {
		err := errors.New("service not found")
		c.logger.Error("meta service not found", zap.Error(err))
		return err
	}
	target := instances[rand.Intn(len(instances))]
	connect, err := grpc.Dial(utils.AddressOffsetRPC(target), grpc.WithInsecure())
	if err != nil {
		c.logger.Error("grpc dial to meta node error", zap.Error(err))
		return err
	}
	defer connect.Close()
	metaClient := meta_service.NewMetaServiceClient(connect)
	objectResp, err := metaClient.QueryObjectMeta(context.TODO(), &meta_service.QueryObjectMetaReq{
		UniqueKey: uniqueKey,
	})
	if err != nil {
		c.logger.Error("meta service query object failed", zap.Error(err))
		return err
	}
	objectMeta := objectResp.Meta
	if objectMeta == nil {
		c.logger.Info("object status not allow delete", zap.String("key", uniqueKey))
		return nil
	}
	_, err = metaClient.UpdateObjectStatus(context.TODO(), &meta_service.UpdateObjectStatusReq{
		UniqueKey: uniqueKey,
		Status:    meta_service.EntryStatus_Deleted,
	})
	if err != nil {
		c.logger.Error("object meta status update error", zap.Error(err))
		return err
	}

	var (
		limit       = 100
		begin       = 0
		end         = int(math.Min(float64(begin+limit), float64(len(objectMeta.ChunkList))))
		chunkMetas  = make([]*meta_service.ChunkMeta, 0, len(objectMeta.ChunkList))
		storeGroups = make(map[int64]struct{}, 0)
	)
	for {
		chunkResp, err := metaClient.QueryChunkMeta(context.TODO(), &meta_service.QueryChunkMetaReq{
			UniqueKeys: objectMeta.ChunkList[begin:end],
		})
		if err != nil || len(chunkResp.Result) != end-begin {
			c.logger.Error("meta service query chunk failed", zap.Error(err))
			return err
		}
		for _, chunkMeta := range chunkResp.Result {
			chunkMetas = append(chunkMetas, chunkMeta)
			storeGroups[chunkMeta.StoreGroup] = struct{}{}
		}
		_, err = metaClient.UpdateChunkStatus(context.TODO(), &meta_service.UpdateChunkStatusReq{
			UniqueKeys: objectMeta.ChunkList[begin:end],
			Status:     meta_service.EntryStatus_Deleted,
		})
		if err != nil {
			c.logger.Error("chunk meta status update error", zap.Error(err))
			return err
		}
		if len(objectMeta.ChunkList) < limit {
			break
		}
		begin = begin + limit
		end = int(math.Min(float64(end+limit), float64(len(objectMeta.ChunkList))))
	}

	storeAddress := make(map[int64][]string, 0)
	for storeGroup := range storeGroups {
		addressResp, err := metaClient.QueryStorageAddress(context.TODO(), &meta_service.QueryStorageAddressReq{
			StoreGroup: storeGroup,
			FromMaster: false,
		})
		if err != nil {
			c.logger.Error("query chunk storage address failed", zap.Error(err))
			return err
		}
		storeAddress[storeGroup] = addressResp.AddressList
	}

	var waitGroup sync.WaitGroup
	for _, chunkMeta := range chunkMetas {
		if ok := utils.PathExist(c.generateChunkPath(chunkMeta.UniqueKey, chunkMeta.Parent)); ok {
			continue
		}
		waitGroup.Add(1)
		go func(addressList []string, meta *meta_service.ChunkMeta) {
			defer waitGroup.Done()
			err := c.deleteChunkBinaryData(addressList, meta)
			if err != nil {
				c.logger.Error("concurrent delete chunk failed", zap.Error(err))
				return
			}
			c.logger.Info("chunk has been delete", zap.String("key", meta.UniqueKey))
		}(storeAddress[chunkMeta.StoreGroup], chunkMeta)
	}
	waitGroup.Wait()

	return nil
}

// deleteChunkBinaryData delete the chunk file with it unique key and address
// in block storage service. (Concurrency Safe)
func (c *DefaultClient) deleteChunkBinaryData(addressList []string, meta *meta_service.ChunkMeta) error {
	address, _ := url.Parse(addressList[rand.Intn(len(addressList))])
	connect, err := grpc.Dial(utils.AddressOffsetRPC(*address), grpc.WithInsecure())
	if err != nil {
		c.logger.Error("grpc dial to block node error", zap.Error(err))
		return err
	}
	defer connect.Close()
	blockClient := block_service.NewBlockServiceClient(connect)
	_, err = blockClient.DeleteBlock(context.TODO(), &block_service.DeleteBlockReq{
		UniqueKey: meta.UniqueKey,
	})
	if err != nil {
		c.logger.Error("block service delete chunk failed", zap.Error(err), zap.String("key", meta.UniqueKey))
		return err
	}
	return nil
}

func (c *DefaultClient) Query(name string, page, pageSize int64) ([]string, int64, error) {
	prefix := kvstore.GenerateObjectPrefix(name)
	instances := service.DiscoveryClient.Instances("shamrock-meta")
	if len(instances) == 0 {
		err := errors.New("service not found")
		c.logger.Error("meta service not found", zap.Error(err))
		return nil, 0, err
	}
	target := instances[rand.Intn(len(instances))]
	connect, err := grpc.Dial(utils.AddressOffsetRPC(target), grpc.WithInsecure())
	if err != nil {
		c.logger.Error("grpc dial to meta node error", zap.Error(err))
		return nil, 0, err
	}
	defer connect.Close()
	metaClient := meta_service.NewMetaServiceClient(connect)
	objectResp, err := metaClient.QueryObjectKeys(context.TODO(), &meta_service.QueryObjectKeysReq{
		Prefix:   prefix,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.logger.Error("page load object keys error", zap.Error(err))
		return nil, 0, err
	}
	return objectResp.UniqueKeys, objectResp.Total, nil
}
