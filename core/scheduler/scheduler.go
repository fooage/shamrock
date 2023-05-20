package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/fooage/shamrock/proto/proto_gen/block_service"
	"github.com/fooage/shamrock/service"
	"github.com/fooage/shamrock/utils"
	"go.uber.org/zap"
)

type Scheduler interface {
	// Dispatch function is to find the right storage grouping for storage space needs.
	Dispatch(size int64) (int64, error)
	// The Proxy function returns the address of the block storage instance that
	// meets the matching parameter.
	Proxy(group int64, fromMaster bool) ([]string, error)
}

type storageScheduler struct {
	healthStatus map[int64]map[int64]*block_service.HealthReport
	requestGroup sync.WaitGroup
	mutex        sync.RWMutex
	logger       *zap.Logger
}

func NewScheduler(logger *zap.Logger) Scheduler {
	scheduler := &storageScheduler{
		healthStatus: make(map[int64]map[int64]*block_service.HealthReport),
		logger:       logger,
	}
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			scheduler.pollingHealthStatus()
			<-ticker.C
		}
	}()
	return scheduler
}

// pollingHealthStatus according to the current list of block storage
// instances, the health status of all nodes is requested.
func (ds *storageScheduler) pollingHealthStatus() {
	instances := service.DiscoveryClient.Instances("shamrock-block")
	healthStatus := make(map[int64]map[int64]*block_service.HealthReport)

	// concurrently request all node health states write in new map
	ds.requestGroup.Add(len(instances))
	for _, address := range instances {
		go func(url url.URL) {
			defer ds.requestGroup.Done()
			report, _ := ds.requestHealthStatus(url)
			if report != nil {
				ds.mutex.Lock()
				if _, ok := healthStatus[report.StoreGroup]; !ok {
					healthStatus[report.StoreGroup] = make(map[int64]*block_service.HealthReport)
				}
				healthStatus[report.StoreGroup][report.StoreNode] = report
				ds.mutex.Unlock()
			}
		}(address)
	}
	ds.requestGroup.Wait()

	// replace the old health status map with new one
	ds.mutex.Lock()
	ds.healthStatus = healthStatus
	ds.mutex.Unlock()
}

func (ds *storageScheduler) requestHealthStatus(url url.URL) (*block_service.HealthReport, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/service/health", utils.AddressOffsetHTTP(url)))
	if err != nil {
		ds.logger.Error("request health http get error", zap.Error(err), zap.String("url", url.String()))
		return nil, err
	}
	defer resp.Body.Close()
	var report block_service.HealthReport
	data, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(data, &report)
	if err != nil {
		ds.logger.Error("json decode health report failed", zap.Error(err))
		return nil, err
	}
	return &report, nil
}

func (ds *storageScheduler) Dispatch(size int64) (int64, error) {
	var suitableGroup int64
	var biggestSpace float64
	ds.mutex.RLock()
	// Find the grouping that has the most space remaining and can hold new slices.
	for index, group := range ds.healthStatus {
		groupLimit := math.MaxFloat64
		for _, node := range group {
			groupLimit = math.Min(groupLimit, float64(node.Capacity))
		}
		groupSpace := math.MaxFloat64
		for _, node := range group {
			groupSpace = math.Min(groupSpace, groupLimit-float64(node.CapacityUsed))
		}
		if groupSpace > biggestSpace && groupSpace > float64(size) {
			suitableGroup = index
			biggestSpace = groupSpace
		}
	}
	ds.mutex.RUnlock()
	if suitableGroup == 0 {
		return 0, errors.New("could not find suitable group")
	}
	return suitableGroup, nil
}

func (ds *storageScheduler) Proxy(group int64, fromMaster bool) ([]string, error) {
	health, ok := ds.healthStatus[group]
	if !ok {
		return nil, errors.New("matched node not found")
	}
	matched := make([]string, 0, len(health))
	for _, instance := range health {
		// If the leader node is obtained, the slave node is filtered.
		if fromMaster && !instance.IsLeader {
			continue
		}
		matched = append(matched, instance.Address)
	}
	return matched, nil
}
