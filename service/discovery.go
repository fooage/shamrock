package service

import (
	"net/url"

	"go.uber.org/zap"
)

type Discovery interface {
	Register(name string, url url.URL)
	Deregister(name string)
}

var DiscoveryClient Discovery

// InitServiceDiscovery function, objects are generated in factory like mode, which is convenient for adding
// other kinds of service discovery components later to adapt to various deployment environments.
func InitServiceDiscovery(logger *zap.Logger, module string, targets []string) Discovery {
	switch module {
	case "consul":
		DiscoveryClient = initConsulDiscovery(logger, targets[0])
	default:
		logger.Panic("not support module of service discovery")
	}
	return DiscoveryClient
}
