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
	return Node{address: address, reputationGood: 5, credibilityGood: 5} // all other fields will be zero by default
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

type probeRecommender struct {
	address          string
	quarantinedProbe bool
}

type Probe struct {
	Address   string
	Reputable bool

	recommenders []probeRecommender
}

// Are we sure that we want reputation manager to pick nodes for us? Maybe this should be moved to the client?
func (rm *ReputationManager) GetProbes(sender string) []Probe {
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

	probesMap := make(map[string]Probe)

	for recType := 0; recType < len(cntRec); recType++ {
		indPerm := rand.Perm(len(recommenders[recType]))
		for pos := 0; pos < cntRec[recType] && pos < len(recommenders[recType]); pos++ {
			ind := indPerm[pos]
			recommender := recommenders[recType][ind]
			conn, err := grpc.Dial(recommender.address, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect to %s: %v", recommender.address, err)
				// if the request to a credible recommender fails we want to find another one to not lose the voting process quality
				if recType == 0 {
					cntRec[recType] += 1
				}
			}
			defer conn.Close()
			c := pb.NewReputationClient(conn)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			r, err := c.GetProbes(ctx, &pb.GetProbesRequest{Sender: sender})
			if err != nil {
				log.Printf("error during recommender request to %s: %v", recommender.address, err)
				// if the request to a credible recommender fails we want to find another one to not lose the voting process quality
				if recType == 0 {
					cntRec[recType] += 1
				}
			}

			for _, probeMsg := range r.GetProbes() {
				if probeMsg.GetAddress() == sender {
					continue
				}
				probe := Node{address: probeMsg.GetAddress(), reputationGood: int(probeMsg.GetReputationGood()), reputationBad: int(probeMsg.ReputationBad)}
				node, ok := probesMap[probe.address]
				if !ok {
					node.Address = probe.address
					node.recommenders = make([]probeRecommender, 0)
				}
				if IsReputable(probe) { // reputable (for recommender) probe
					if recType == 0 { // credible recommender
						node.Reputable = true // we trust recommender
						node.recommenders = append(node.recommenders, probeRecommender{address: recommender.address, quarantinedProbe: false})
					} else { // quarantined recommender
						node.recommenders = append(node.recommenders, probeRecommender{address: recommender.address, quarantinedProbe: false})
					}
				} else { // quarantined (for recommender) probe
					if recType == 0 { // credible recommender
						node.recommenders = append(node.recommenders, probeRecommender{address: recommender.address, quarantinedProbe: true})
					}
					// we don't need quarantined probes from quarantined recommenders
				}
				probesMap[probe.address] = node
			}
		}
	}
	probes := make([]Probe, 0)
	probesQuarantine := make([]Probe, 0)

	for _, probe := range probesMap {
		if probe.Reputable {
			probes = append(probes, probe)
		} else {
			probesQuarantine = append(probesQuarantine, probe)
		}
	}

	cntProbes := pickProbes
	if len(probes) < cntProbes {
		cntProbes = len(probes)
	}
	cntProbesQ := pickProbesQuarantine
	if len(probesQuarantine) < cntProbesQ {
		cntProbesQ = len(probesQuarantine)
	}

	res := make([]Probe, 0, cntProbes+cntProbesQ)

	for _, i := range pickN(len(probes), cntProbes) {
		res = append(res, probes[i])
	}
	for _, i := range pickN(len(probesQuarantine), cntProbesQ) {
		res = append(res, probes[i])
	}

	return res
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
func (rm *ReputationManager) EvaluateVotes(probes []Probe, satisfaction []int) {
	for i, probe := range probes {
		if satisfaction[i] > 0 {
			rm.mutex.Lock()
			rm.Nodes[probe.Address] = RaiseReputation(getDefault(rm.Nodes, probe.Address))
			for _, recommender := range probe.recommenders {
				if !recommender.quarantinedProbe { // if a good probe was also reputable in recommender's point of view then it probably is credible
					rm.Nodes[recommender.address] = RaiseCredibility(getDefault(rm.Nodes, recommender.address))
				}
			}
			rm.mutex.Unlock()
		} else if satisfaction[i] < 0 {
			rm.mutex.Lock()
			rm.Nodes[probe.Address] = LowerReputation(getDefault(rm.Nodes, probe.Address))
			for _, recommender := range probe.recommenders {
				if !recommender.quarantinedProbe { // // if a bad probe was reputable in recommender's point of view then it probably is not credible
					rm.Nodes[recommender.address] = LowerCredibility(getDefault(rm.Nodes, recommender.address))
				}
			}
			rm.mutex.Unlock()
		}
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
