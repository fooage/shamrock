package kvstore

import "fmt"

const (
	ObjectMetaKeyTemplate string = "object-meta:%s"
	ChunkMetaKeyTemplate  string = "chunk-meta:%s"
)

func GenerateObjectMetaKey(name string) string {
	return fmt.Sprintf(ObjectMetaKeyTemplate, name)
}

func GenerateChunkMetaKey(hash string) string {
	return fmt.Sprintf(ChunkMetaKeyTemplate, hash)
}
