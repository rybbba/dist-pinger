package client

import (
	"context"
	"log" // TODO: remove log from client code
	"time"

	pb "github.com/rybbba/dist-pinger/grpc"
	"github.com/rybbba/dist-pinger/reputation"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Node struct {
	address string
}

type PingerClient struct {
	RepManager *reputation.ReputationManager
	id         string
	addrs      []string
	nodes      map[string]Node
}

func (pingerClient *PingerClient) SetId(id string) {
	pingerClient.id = id
}

func (pingerClient *PingerClient) SetNodes(addrs []string) {
	pingerClient.addrs = addrs
	pingerClient.nodes = make(map[string]Node)
	for _, addr := range addrs {
		pingerClient.nodes[addr] = Node{address: addr}
	}
}

func (pingerClient *PingerClient) GetStatus(host string) {
	probesB := pingerClient.RepManager.GetProbes(pingerClient.id) // reputable and quarantined probes

	results := make([]int32, 0, len(probesB))
	resultsToPrint := make([]int32, 0)
	aggResults := make(map[int32]int)

	var bestAns int32 = 0
	for _, probe := range probesB {
		if probe.Reputable {
			log.Printf("Using probe: %v", probe.Address)
		} else {
			log.Printf("Using quarantined probe: %v", probe.Address)
		}

		conn, err := grpc.Dial(probe.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()
		c := pb.NewPingerClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		r, err := c.CheckHost(ctx, &pb.CheckHostRequest{Host: host, Sender: pingerClient.id}) // TODO: The whole id thing is a big crutch right now that should be removed
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

	satisfied := make([]int, 0, len(results))
	for _, code := range results {
		if code == bestAns {
			satisfied = append(satisfied, 1)
		} else {
			satisfied = append(satisfied, -1)
		}
	}

	pingerClient.RepManager.EvaluateVotes(probesB, satisfied) // append ruins probes[0]

	for _, probe := range probesB {
		log.Print(pingerClient.RepManager.Nodes[probe.Address])
	}

	log.Printf("Check result for host %s: %v", host, resultsToPrint) // only print results by reputable probes
	log.Printf("Aggregated results: %v", aggResults)
	log.Printf("Resource status: %d", bestAns)
	// TODO: Deal with multiple "right" answers (geo-specific access restriction, etc.)
}
