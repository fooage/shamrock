package kvstore

import (
	"fmt"

	"github.com/fooage/shamrock/utils"
)

const (
	objectUniqueKeyTemplate string = "object-%s-%d"
	chunkUniqueKeyTemplate  string = "chunk-%s-%d"
)

var generator = utils.NewGenerator(0, 0)

func GenerateObjectKey(name string) string {
	return fmt.Sprintf(objectUniqueKeyTemplate, name, generator.NextVal())
}

func GenerateChunkKey(hash string) string {
	return fmt.Sprintf(chunkUniqueKeyTemplate, hash, generator.NextVal())
}
