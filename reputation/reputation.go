package reputation

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	pb "github.com/rybbba/dist-pinger/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	pickProbes       = 3
	pickRecommenders = 2

	// what rating formula will we use?
	reputationThreshold  = 2
	credibilityThreshold = 2

	pickProbesQuarantine       = 2
	pickRecommendersQuarantine = 1
)

type Node struct {
	address         string
	reputationGood  int
	reputationBad   int
	credibilityGood int
	credibilityBad  int
}

func nodeInit(address string) Node {
	return Node{address: address} // all other fields will be zero by default
}

func IsReputable(node Node) bool {
	return node.reputationGood-node.reputationBad >= reputationThreshold
}

func RaiseReputation(node Node) Node {
	node.reputationGood += 1
	return node
}

func LowerReputation(node Node) Node {
	node.reputationBad += 1
	return node
}

func IsCredible(node Node) bool {
	return node.credibilityGood-node.credibilityBad >= credibilityThreshold
}

func RaiseCredibility(node Node) Node {
	node.credibilityGood += 1
	return node
}

func LowerCredibility(node Node) Node {
	node.credibilityBad += 1
	return node
}

type ReputationManager struct {
	Nodes map[string]Node

	mutex sync.RWMutex
}

func (rm *ReputationManager) InitZeros(addrs []string) {
	rm.Nodes = make(map[string]Node)
	for _, addr := range addrs {
		rm.Nodes[addr] = nodeInit(addr)
	}
}

func pickN(total int, n int) []int {
	return rand.Perm(total)[:n]
}

// TODO: make sure that following functions will work as intended
// with an address that is not in the manager's nodes keys

// TODO: I 100% must refactor this
func (rm *ReputationManager) GiveProbes() *pb.GetProbesResponse {
	message := &pb.GetProbesResponse{Probes: make([]*pb.Probe, 0)}
	for _, node := range rm.Nodes {
		message.Probes = append(message.Probes, &pb.Probe{Address: node.address, ReputationGood: int32(node.reputationGood), ReputationBad: int32(node.reputationBad)})
	}
	return message
}

type PickedProbe struct {
	Address       string
	isQuarantined bool

	recommenderAddress     string
	recommenderQuarantined bool
}

// Are we sure that we want reputation manager to pick nodes for us? Maybe this should be moved to the client?
func (rm *ReputationManager) GetProbes(sender string) ([]PickedProbe, []PickedProbe) {
	rm.mutex.RLock()
	recommenders := [][]Node{make([]Node, 0), make([]Node, 0)} // reliable and quarantined

	for _, node := range rm.Nodes {
		if IsCredible(node) {
			recommenders[0] = append(recommenders[0], node)
		} else {
			recommenders[1] = append(recommenders[1], node)
		}
	}
	rm.mutex.RUnlock()

	cntRec := []int{pickRecommenders, pickRecommendersQuarantine} // for reliable and quarantined random picks
	for i := 0; i < len(cntRec); i++ {
		if len(recommenders[i]) < cntRec[i] {
			cntRec[i] = len(recommenders[i])
		}
	}

	probes := make([]PickedProbe, 0)
	probesQuarantine := make([]PickedProbe, 0)

	for i := 0; i < len(cntRec); i++ {
		for _, ind := range pickN(len(recommenders), cntRec[i]) {
			recommender := recommenders[i][ind]
			conn, err := grpc.Dial(recommender.address, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect: %v", err)
				continue // TODO: For now we will ignore if recommender is unreachable, this behaviour should be changed
			}
			defer conn.Close()
			c := pb.NewReputationClient(conn)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			r, err := c.GetProbes(ctx, &pb.GetProbesRequest{Sender: sender})
			if err != nil {
				log.Printf("error during recommender request: %v", err)
				continue // TODO: this should be replaced
			}

			for _, probe := range r.GetProbes() {
				if probe.GetAddress() == sender {
					continue
				}
				node := Node{address: probe.GetAddress(), reputationGood: int(probe.GetReputationGood()), reputationBad: int(probe.ReputationBad)}
				if IsReputable(node) {
					if i == 0 { // reliable recommender
						probes = append(probes, PickedProbe{Address: node.address, isQuarantined: false,
							recommenderAddress: recommender.address, recommenderQuarantined: false}) // reliable recommender, reputable probe
					} else { // quarantined recommender
						probesQuarantine = append(probesQuarantine, PickedProbe{Address: node.address, isQuarantined: false,
							recommenderAddress: recommender.address, recommenderQuarantined: true}) // quarantined recommender, reputable probe
					}
				} else {
					if i == 0 { // reliable recommender
						probesQuarantine = append(probes, PickedProbe{Address: node.address, isQuarantined: true,
							recommenderAddress: recommender.address, recommenderQuarantined: false}) // reliable recommender, quarantined probe
					}
					// we don't need quarantined probes from quarantined recommenders
				}
			}
		}
	}
	// Now there could be duplicates within one probe slice or even between probes and probesQuarantine
	// We will deduplicate them leaving probes only outside quarantine if present in both slices

	// TODO: at the moment we are removing duplicates and at the same time removing recommenders of these duplicates, that's not how it should work
	dedupProbes := make(map[string]bool)
	probesD := make([]PickedProbe, 0)
	for _, probe := range probes {
		if !dedupProbes[probe.Address] {
			probesD = append(probesD, probe)
			dedupProbes[probe.Address] = true
		}
	}
	probes = probesD
	probesD = make([]PickedProbe, 0)
	for _, probe := range probesQuarantine {
		if !dedupProbes[probe.Address] {
			probesD = append(probesD, probe)
			dedupProbes[probe.Address] = true
		}
	}
	probesQuarantine = probesD

	cntProbes := pickProbes
	if len(probes) < cntProbes {
		cntProbes = len(probes)
	}
	cntProbesQ := pickProbesQuarantine
	if len(probesQuarantine) < cntProbesQ {
		cntProbesQ = len(probesQuarantine)
	}

	res := make([]PickedProbe, 0, cntProbes)
	resQ := make([]PickedProbe, 0, cntProbesQ)

	for _, i := range pickN(len(probes), cntProbes) {
		res = append(res, probes[i])
	}
	for _, i := range pickN(len(probesQuarantine), cntProbesQ) {
		resQ = append(resQ, probesQuarantine[i])
	}

	return res, resQ
}

func getDefault(m map[string]Node, address string) Node {
	node, ok := m[address]
	if !ok {
		return nodeInit(address)
	}
	return node
}

// Takes an array of probes returned by GetServers and an array of our satisfaction from corresponding probes' work
// Negative satisfaction means that probe's answer was bad, positive that it was good and zero means that we don't want to rate it
func (rm *ReputationManager) EvaluateVotes(probes []PickedProbe, satisfaction []int) {
	for i, probe := range probes {
		if satisfaction[i] > 0 {
			rm.mutex.Lock()
			rm.Nodes[probe.Address] = RaiseReputation(getDefault(rm.Nodes, probe.Address))
			if !probe.isQuarantined { // if a good probe was also reputable in recommender's point then he probably is credible
				rm.Nodes[probe.recommenderAddress] = RaiseCredibility(getDefault(rm.Nodes, probe.recommenderAddress))
			}
			rm.mutex.Unlock()
		} else if satisfaction[i] < 0 {
			rm.mutex.Lock()
			rm.Nodes[probe.Address] = LowerReputation(getDefault(rm.Nodes, probe.Address))
			if !probe.isQuarantined { // if a bad probe was reputable in recommender's point then he probably is not credible
				rm.Nodes[probe.recommenderAddress] = LowerCredibility(getDefault(rm.Nodes, probe.recommenderAddress))
			}
			rm.mutex.Unlock()
		}
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}