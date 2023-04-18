package http_api

import "fmt"

const (
	ObjectMetaKeyTemplate string = "object-meta:%s"
	ChunkMetaKeyTemplate  string = "chunk-meta:%s"
)

func generateObjectMetaKey(name string) string {
	return fmt.Sprintf(ObjectMetaKeyTemplate, name)
}

func generateChunkMetaKey(hash string) string {
	return fmt.Sprintf(ChunkMetaKeyTemplate, hash)
}
