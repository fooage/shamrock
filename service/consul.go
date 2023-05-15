package service

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/fooage/shamrock/utils"

	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

type consulClient struct {
	client *api.Client
	logger *zap.Logger
}

func initConsulDiscovery(logger *zap.Logger, target string) *consulClient {
	config := api.DefaultConfig()
	config.Address = target
	config.WaitTime = 5 * time.Second
	client, err := api.NewClient(config)
	if err != nil {
		logger.Panic("create consul register error", zap.Error(err))
		return nil
	}
	return &consulClient{
		client: client,
		logger: logger,
	}
}

// Register is service discovery registration implemented by Consul.
func (c *consulClient) Register(name string, url url.URL) {
	host, portStr, _ := net.SplitHostPort(utils.AddressOffsetHTTP(url))
	port, _ := strconv.Atoi(portStr)
	registration := new(api.AgentServiceRegistration)
	registration.Name = name
	registration.ID = fmt.Sprintf("%s:%d", host, port)
	registration.Address = host
	registration.Port = port
	// The service health monitoring use the http API interface, so an address
	// transform (AddressOffsetHTTP) is done on it.
	registration.Check = &api.AgentServiceCheck{
		Interval:                       "10s",
		HTTP:                           fmt.Sprintf("http://%s:%d/service/health", host, port),
		Timeout:                        "5s",
		DeregisterCriticalServiceAfter: "30s",
	}
	err := c.client.Agent().ServiceRegister(registration)
	if err != nil {
		c.logger.Panic("service could not be register", zap.Error(err))
		return
	}
}

// Deregister is service logout deregister base on the Consul implementation.
func (c *consulClient) Deregister(name string) {
	err := c.client.Agent().ServiceDeregister(name)
	if err != nil {
		c.logger.Panic("service could not be deregister", zap.Error(err))
		return
	}
}
