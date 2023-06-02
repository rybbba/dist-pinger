package reputation

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	pb "github.com/rybbba/dist-pinger/grpc"
	"github.com/rybbba/dist-pinger/identity"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	pickRecommenders = 2

	pickProbesQuarantine       = 2
	pickRecommendersQuarantine = 1
)

type ReputationManager struct {
	Nodes map[string]Node

	mutex sync.RWMutex
}

// debug output function
func (rm *ReputationManager) PrintSimpleRep() string {
	res := ""
	rm.mutex.Lock()
	first := true
	for _, node := range rm.Nodes {
		if !first {
			res += ","
		}
		first = false
		res += fmt.Sprintf("%s: %d %d", node.user.Address, node.reputationGood-node.reputationBad, node.credibilityGood-node.credibilityBad)
	}
	rm.mutex.Unlock()
	return res
}

func (rm *ReputationManager) InitNodes(users []identity.PublicUser) {
	rm.Nodes = make(map[string]Node)
	for _, user := range users {
		rm.Nodes[user.Id] = nodeInitRef(user)
	}
}

func pickN(total int, n int) []int {
	return rand.Perm(total)[:n]
}

// TODO: make sure that following functions will work as intended
// with an address that is not in the manager's nodes keys

// TODO: I must refactor this
func (rm *ReputationManager) GiveProbes(sender identity.PublicUser, withCredibility bool) *pb.GetReputationsResponse {
	message := &pb.GetReputationsResponse{Probes: make([]*pb.Probe, 0)}
	for _, node := range rm.Nodes {
		probeMsg := pb.Probe{Id: node.user.Id, ReputationGood: int32(node.reputationGood), ReputationBad: int32(node.reputationBad)}
		if withCredibility {
			probeMsg.CredibilityGood = int32(node.credibilityGood)
			probeMsg.CredibilityBad = int32(node.credibilityBad)
		}
		message.Probes = append(message.Probes, &probeMsg)
	}
	if _, ok := rm.Nodes[sender.Id]; !ok {
		rm.Nodes[sender.Id] = nodeInit(sender)
	}
	return message
}

func (rm *ReputationManager) CopyReputation(sender identity.PrivateUser, target identity.PublicUser) error {
	conn, err := grpc.Dial(target.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	c := pb.NewReputationClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	message := pb.GetReputationsRequest{Sender: sender.Id, NeedCredibilities: true}
	signature, err := identity.SignProto(sender, &message)
	if err != nil {
		log.Fatalf("cannot sign message: %v", err)
	}
	message.Signature = signature

	r, err := c.GetReputations(ctx, &message)
	if err != nil {
		return err
	}

	for _, probeMsg := range r.GetProbes() {
		if probeMsg.Id == sender.Id {
			continue
		}

		nodeUser, err := identity.ParseUser(probeMsg.Id)
		if err != nil {
			continue
		}
		node := nodeInit(nodeUser)
		node.reputationGood, node.reputationBad = int(probeMsg.ReputationGood), int(probeMsg.ReputationBad)
		node.credibilityGood, node.credibilityBad = int(probeMsg.CredibilityGood), int(probeMsg.CredibilityBad)
		rm.mutex.Lock()
		rm.Nodes[nodeUser.Id] = node
		rm.mutex.Unlock()
	}
	rm.Nodes[target.Id] = nodeInitRef(target)
	// log.Printf("Copied reputations: %v", rm.Nodes)
	return nil
}

// Are we sure that we want reputation manager to pick nodes for us? Maybe this should be moved to the client?
func (rm *ReputationManager) GetProbes(sender identity.PrivateUser, pickProbes int) []Probe {
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
			conn, err := grpc.Dial(recommender.user.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("did not connect to %s: %v", recommender.user.Address, err)
				// if the request to a credible recommender fails we want to find another one to not lose the voting process quality
				if recType == 0 {
					cntRec[recType] += 1
				}
			}
			defer conn.Close()
			c := pb.NewReputationClient(conn)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			message := pb.GetReputationsRequest{Sender: sender.Id}
			signature, err := identity.SignProto(sender, &message)
			if err != nil {
				log.Fatalf("cannot sign message: %v", err)
			}
			message.Signature = signature

			r, err := c.GetReputations(ctx, &message)
			if err != nil {
				log.Printf("error during recommender request to %s: %v", recommender.user.Address, err)
				// if the request to a credible recommender fails we want to find another one to not lose the voting process quality
				if recType == 0 {
					cntRec[recType] += 1
				}
			}

			for _, probeMsg := range r.GetProbes() {
				if probeMsg.GetId() == sender.Id {
					continue
				}
				probeUser, err := identity.ParseUser(probeMsg.Id)
				if err != nil {
					continue
				}
				if _, ok := rm.Nodes[probeUser.Id]; !ok {
					rm.Nodes[probeUser.Id] = nodeInit(probeUser)
				}

				node := Node{user: probeUser, reputationGood: int(probeMsg.GetReputationGood()), reputationBad: int(probeMsg.ReputationBad)}
				probe, ok := probesMap[node.user.Id]
				if !ok {
					probe.User = node.user
					probe.recommenders = make([]probeRecommender, 0)
				}
				if IsReputable(node) { // reputable (for recommender) probe
					if recType == 0 { // credible recommender
						probe.Reputable = true // we trust recommender
						probe.recommenders = append(probe.recommenders, probeRecommender{user: recommender.user, quarantinedProbe: false})
					} else { // quarantined recommender
						probe.recommenders = append(probe.recommenders, probeRecommender{user: recommender.user, quarantinedProbe: false})
					}
				} else { // quarantined (for recommender) probe
					if recType == 0 { // credible recommender
						probe.recommenders = append(probe.recommenders, probeRecommender{user: recommender.user, quarantinedProbe: true})
					}
					// we don't need quarantined probes from quarantined recommenders
				}
				probesMap[probe.User.Id] = probe
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
		res = append(res, probesQuarantine[i])
	}

	return res
}

// Takes an array of probes returned by GetServers and an array of our satisfaction from corresponding probes' work
// Negative satisfaction means that probe's answer was bad, positive that it was good and zero means that we don't want to rate it
// All passed probes already have a record in rm.Nodes
func (rm *ReputationManager) EvaluateVotes(probes []Probe, satisfaction []int) {
	for i, probe := range probes {
		if satisfaction[i] > 0 {
			rm.mutex.Lock()
			rm.Nodes[probe.User.Id] = RaiseReputation(rm.Nodes[probe.User.Id])
			for _, recommender := range probe.recommenders {
				if !recommender.quarantinedProbe { // if a good probe was also reputable in recommender's point of view then it probably is credible
					rm.Nodes[recommender.user.Id] = RaiseCredibility(rm.Nodes[recommender.user.Id])
				}
			}
			rm.mutex.Unlock()
		} else if satisfaction[i] < 0 {
			rm.mutex.Lock()
			rm.Nodes[probe.User.Id] = LowerReputation(rm.Nodes[probe.User.Id])
			for _, recommender := range probe.recommenders {
				if !recommender.quarantinedProbe { // // if a bad probe was reputable in recommender's point of view then it probably is not credible
					rm.Nodes[recommender.user.Id] = LowerCredibility(rm.Nodes[recommender.user.Id])
				}
			}
			rm.mutex.Unlock()
		}
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
