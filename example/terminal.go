package main

import (
	"flag"

	"github.com/fooage/shamrock/api/client"
	"go.uber.org/zap"
)

var (
	operate   *string
	path      *string
	prefix    *string
	key       *string
	discovery *string
)

func init() {
	operate = flag.String("operate", "", "client's operate to system")
	path = flag.String("path", "", "target path of file to operate")
	prefix = flag.String("prefix", "", "query prefix word to find object keys")
	key = flag.String("key", "", "unique key of operate object")
	discovery = flag.String("discovery", "127.0.0.1:8500", "separated service discovery")
	flag.Parse()
}

func main() {
	logger, _ := zap.NewProduction()
	client := client.NewDefaultClient(logger, []string{*discovery})
	switch *operate {
	case "fetch":
		local, err := client.Fetch(*key)
		if err != nil {
			logger.Error("client fetch object error", zap.Error(err))
			return
		}
		logger.Info("local file path", zap.String("path", local))
	case "register":
		key, err := client.Register(*path)
		if err != nil {
			logger.Error("client register object error", zap.Error(err))
			return
		}
		logger.Info("object register success", zap.String("key", key))
	case "store":
		err := client.Store(*key, *path)
		if err != nil {
			logger.Error("client store object error", zap.Error(err))
			return
		}
		logger.Info("store object done", zap.String("key", *key))
	case "delete":
		err := client.Delete(*key)
		if err != nil {
			logger.Error("client delete object error", zap.Error(err))
			return
		}
		logger.Info("object has been deleted", zap.String("key", *key))
	case "query":
		keys, _, err := client.Query(*prefix, 0, 100)
		if err != nil {
			logger.Error("client query object error", zap.Error(err))
			return
		}
		logger.Info("object query result", zap.Strings("keys", keys))
	}
}
