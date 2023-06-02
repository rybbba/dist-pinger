package client

import (
	"context"
	"log"
	"time"

	pb "github.com/rybbba/dist-pinger/grpc"
	"github.com/rybbba/dist-pinger/identity"
	"github.com/rybbba/dist-pinger/reputation"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	pickProbes = 3
)

type Node struct {
	address string
}

type PingerClient struct {
	RepManager reputation.ReputationManagerInterface
	user       identity.PrivateUser
}

func (pingerClient *PingerClient) SetUser(user identity.PrivateUser) {
	pingerClient.user = user
}

func (pingerClient *PingerClient) GetStatus(host string) {
	// TODO: At this moment we get exactly pickProbes probes and if some of them don't answer we have fewer probes to vote
	probes := pingerClient.RepManager.GetProbes(pingerClient.user, pickProbes)

	results := make([]int32, 0, len(probes))
	resultsToPrint := make([]int32, 0)
	aggResults := make(map[int32]int)

	var bestAns int32 = 0
	for _, probe := range probes {
		if probe.Reputable {
			log.Printf("Using probe: %v", probe.User.Address)
		} else {
			log.Printf("Using quarantined probe: %v", probe.User.Address)
		}

		conn, err := grpc.Dial(probe.User.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("Cannot not connect: %v", err)
			continue
		}
		defer conn.Close()
		c := pb.NewPingerClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		message := pb.CheckHostRequest{Host: host, Sender: pingerClient.user.Id}
		signature, err := identity.SignProto(pingerClient.user, &message)
		if err != nil {
			log.Fatalf("cannot sign message: %v", err)
		}
		message.Signature = signature

		r, err := c.CheckHost(ctx, &message)
		var code int32
		if err != nil {
			log.Printf("error during probe request: %v", err)
			code = 0
		} else {
			code = r.GetCode()
		}

		results = append(results, code)
		if probe.Reputable { // update best answer if probe is reputable
			resultsToPrint = append(resultsToPrint, code)

			aggResults[code] += 1
			if aggResults[code] > aggResults[bestAns] {
				bestAns = code
			}
		}
	}

	if bestAns != 0 { // At the moment 0 means that some kind of problem was encountered during ping process, we don't want to rate nodes if most of them are faulty
		satisfied := make([]int, 0, len(results))
		for _, code := range results {
			if code == bestAns {
				satisfied = append(satisfied, 1)
			} else {
				satisfied = append(satisfied, -1)
			}
		}

		pingerClient.RepManager.EvaluateVotes(probes, satisfied) // usage of append inside EvaluateVotes will ruin probes[0]
	}

	log.Printf("Check result for host %s: %v", host, resultsToPrint) // only print results by reputable probes
	log.Printf("Aggregated results: %v", aggResults)
	log.Printf("Resource status: %d", bestAns)
}
