package service

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/fooage/shamrock/utils"
	"github.com/hashicorp/consul/api"
	"go.etcd.io/etcd/client/pkg/v3/types"
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
	host, portStr, _ := net.SplitHostPort(url.Host)
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
		HTTP:                           fmt.Sprintf("http://%s/service/health", utils.AddressOffsetHTTP(url)),
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

// Instances find and return the service instance URLs for all matching name.
func (c *consulClient) Instances(name string) types.URLs {
	matched, _, err := c.client.Catalog().Service(name, "", nil)
	if err != nil {
		c.logger.Error("discovery service error", zap.Error(err))
		return nil
	}
	instances := make(types.URLs, 0, len(matched))
	for _, instance := range matched {
		url, err := url.Parse(fmt.Sprintf("http://%s:%d", instance.ServiceAddress, instance.ServicePort))
		if err != nil {
			continue
		}
		instances = append(instances, *url)
	}
	return instances
}
