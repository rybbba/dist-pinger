package reputation

import "sync"

type Node struct {
	address        string
	reputationGood int
	reputationBad  int
}

type ReputationManager struct {
	nodes map[string]Node

	mutex sync.RWMutex
}

func (rm *ReputationManager) Init(addrs []string, rates []int) {
	rm.nodes = make(map[string]Node)
	for i, addr := range addrs {
		rm.nodes[addr] = Node{address: addr, reputationGood: rates[i], reputationBad: 0}
	}
}
func (rm *ReputationManager) InitZeros(addrs []string) {
	rm.nodes = make(map[string]Node)
	for _, addr := range addrs {
		rm.nodes[addr] = Node{address: addr, reputationGood: 0, reputationBad: 0}
	}
}

// TODO: following functions will not work as intended
// with an address that is not in the manager's nodes keys

func (rm *ReputationManager) GetReputation(addr string) int {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	return rm.nodes[addr].reputationGood - rm.nodes[addr].reputationBad
}

func (rm *ReputationManager) IncreaseServer(addr string) {
	rm.mutex.Lock()
	server := rm.nodes[addr]
	server.reputationGood += 1
	rm.nodes[addr] = server
	rm.mutex.Unlock()
}

func (rm *ReputationManager) LowerServer(addr string) {
	rm.mutex.Lock()
	server := rm.nodes[addr]
	server.reputationBad += 1
	rm.nodes[addr] = server
	rm.mutex.Unlock()
}

func (rm *ReputationManager) IncreaseClient(addr string) {
	rm.mutex.Lock()
	client := rm.nodes[addr]
	client.reputationGood += 1
	rm.nodes[addr] = client
	rm.mutex.Unlock()
}

func (rm *ReputationManager) LowerClient(addr string) {
	rm.mutex.Lock()
	client := rm.nodes[addr]
	client.reputationBad += 1
	rm.nodes[addr] = client
	rm.mutex.Unlock()
}
