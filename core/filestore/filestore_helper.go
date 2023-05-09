package filestore

import (
	"fmt"
	"os"

	"github.com/fooage/shamrock/proto/proto_gen/block_service"
	"github.com/fooage/shamrock/utils"
	"go.uber.org/zap"
)

func (f *filestoreServer) GenerateFilePath(name string) string {
	return fmt.Sprintf("%s/%s", f.storePath, name)
}

func (f *filestoreServer) writeBlockLocked(entry *block_service.BlockEntry) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	file, err := os.OpenFile(f.GenerateFilePath(entry.UniqueKey), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		f.logger.Error("file create error", zap.String("file", entry.UniqueKey), zap.Error(err))
		return err
	}
	defer file.Close()
	if err = utils.BufferWriteFile(file, entry.Binary); err != nil {
		f.logger.Error("data write on disk failed", zap.String("file", file.Name()), zap.Error(err))
		return err
	}
	f.filestore[entry.UniqueKey] = &block_service.BlockEntry{
		UniqueKey: entry.UniqueKey,
		Hash:      entry.Hash,
	}
	return nil
}

func (f *filestoreServer) removeBlockLocked(entry *block_service.BlockEntry) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	err := os.Remove(f.GenerateFilePath(entry.UniqueKey))
	if err != nil {
		f.logger.Error("remove file error", zap.String("file", entry.UniqueKey), zap.Error(err))
		return err
	}
	delete(f.filestore, entry.UniqueKey)
	return nil
}
