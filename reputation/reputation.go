package reputation

import "sync"

type Node struct {
	address    string
	reputation int
}

type ReputationManager struct {
	nodes map[string]Node

	mutex sync.RWMutex
}

func (rm *ReputationManager) Init(addrs []string, rates []int) {
	rm.nodes = make(map[string]Node)
	for i, addr := range addrs {
		rm.nodes[addr] = Node{address: addr, reputation: rates[i]}
	}
}
func (rm *ReputationManager) InitZeros(addrs []string) {
	rm.nodes = make(map[string]Node)
	for _, addr := range addrs {
		rm.nodes[addr] = Node{address: addr, reputation: 1}
	}
}

// TODO: following functions will not work as intended
// with an address that is not in the manager's nodes keys

func (rm *ReputationManager) GetReputation(addr string) int {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	return rm.nodes[addr].reputation
}

func (rm *ReputationManager) IncreaseServer(addr string) {
	rm.mutex.Lock()
	server := rm.nodes[addr]
	server.reputation += 1
	rm.nodes[addr] = server
	rm.mutex.Unlock()
}

func (rm *ReputationManager) LowerServer(addr string) {
	rm.mutex.Lock()
	server := rm.nodes[addr]
	server.reputation -= 1
	rm.nodes[addr] = server
	rm.mutex.Unlock()
}

func (rm *ReputationManager) IncreaseClient(addr string) {
	rm.mutex.Lock()
	client := rm.nodes[addr]
	client.reputation += 1
	rm.nodes[addr] = client
	rm.mutex.Unlock()
}

func (rm *ReputationManager) LowerClient(addr string) {
	rm.mutex.Lock()
	client := rm.nodes[addr]
	client.reputation -= 1
	rm.nodes[addr] = client
	rm.mutex.Unlock()
}
