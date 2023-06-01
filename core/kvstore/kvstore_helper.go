package kvstore

import (
	"fmt"
	"strings"

	"github.com/fooage/shamrock/utils"
)

const (
	objectUniqueKeyTemplate    string = "object-%s-%d"
	objectUniquePrefixTemplate string = "object-%s"
	chunkUniqueKeyTemplate     string = "chunk-%s-%d"
	chunkUniquePrefixTemplate  string = "chunk-%s"
)

var generator = utils.NewGenerator(0, 0)

func GenerateObjectKey(name string) string {
	return fmt.Sprintf(objectUniqueKeyTemplate, name, generator.NextVal())
}

func GenerateChunkKey(hash string) string {
	return fmt.Sprintf(chunkUniqueKeyTemplate, hash, generator.NextVal())
}

func GenerateObjectPrefix(name string) string {
	return fmt.Sprintf(objectUniquePrefixTemplate, name)
}

func GenerateChunkPrefix(hash string) string {
	return fmt.Sprintf(chunkUniquePrefixTemplate, hash)
}

func ParseObjectName(key string) string {
	keySplited := strings.Split(key, "-")
	return strings.Join(keySplited[1:len(keySplited)-1], "-")
}

func ParseChunkHash(key string) string {
	return strings.Split(key, "-")[1]
}
