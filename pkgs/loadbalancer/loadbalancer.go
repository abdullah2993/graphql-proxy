package loadbalancer

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/abdullah2993/graphql-proxy/pkgs/config"
)

type LoadBalancer struct {
	mu      sync.RWMutex
	servers []config.UpstreamServer
	weights []int
	total   int
}

func New(servers []config.UpstreamServer) *LoadBalancer {
	weights := make([]int, len(servers))
	total := 0

	for i, server := range servers {
		total += server.Weight
		weights[i] = total
	}

	return &LoadBalancer{
		servers: servers,
		weights: weights,
		total:   total,
	}
}

func (lb *LoadBalancer) GetServer(capability config.Capability, operationName string) (*config.UpstreamServer, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// Filter servers that support the required capability and operation name
	var eligible []config.UpstreamServer
	var eligibleWeights []int
	totalWeight := 0

	for _, server := range lb.servers {
		// Check capabilities
		hasCapability := false
		for _, cap := range server.Capabilities {
			if cap == capability {
				hasCapability = true
				break
			}
		}
		if !hasCapability {
			continue
		}

		// Check operation names
		// If server has no operation names defined, it can handle all operations
		// Otherwise, check if it can handle this specific operation
		if len(server.OperationNames) == 0 || containsOperationName(server.OperationNames, operationName) {
			eligible = append(eligible, server)
			totalWeight += server.Weight
			eligibleWeights = append(eligibleWeights, totalWeight)
		}
	}

	if len(eligible) == 0 {
		return nil, fmt.Errorf("no servers available for capability: %s and operation: %s", capability, operationName)
	}

	// Weighted random selection
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	point := r.Intn(totalWeight)

	for i, weight := range eligibleWeights {
		if point < weight {
			return &eligible[i], nil
		}
	}

	return &eligible[len(eligible)-1], nil
}

func containsOperationName(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}
