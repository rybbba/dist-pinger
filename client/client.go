package client

import (
	"context"
	"errors"
	"log" // TODO: remove log from client code
	"math/rand"
	"time"

	pb "github.com/rybbba/dist-pinger/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Node struct {
	address string
}

func pickN(total int, n int) ([]int, error) {
	if total < n {
		return nil, errors.New("not enough members to pick")
	}
	return rand.Perm(total)[:n], nil
}

type PingerClient struct {
	PickCount int
	addrs     []string
	nodes     map[string]Node
}

func (pingerClient *PingerClient) SetNodes(addrs []string) {
	pingerClient.addrs = addrs
	pingerClient.nodes = make(map[string]Node)
	for _, addr := range addrs {
		pingerClient.nodes[addr] = Node{address: addr}
	}
}

func (pingerClient *PingerClient) GetStatus(host string) {
	using, err := pickN(len(pingerClient.nodes), pingerClient.PickCount)
	if err != nil {
		log.Fatalf("error while picking nodes: %v", err)
	}
	results := make([]int32, pingerClient.PickCount)
	aggResults := make(map[int32]int)
	var bestAns int32 = 0
	for i, nodeInd := range using {
		node := pingerClient.nodes[pingerClient.addrs[nodeInd]]
		log.Printf("Using node: %v", node)

		conn, err := grpc.Dial(node.address, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()
		c := pb.NewPingerClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		r, err := c.CheckHost(ctx, &pb.CheckHostRequest{Host: host})
		var code int32
		if err != nil {
			code = 0
		} else {
			code = r.GetCode()
		}
		aggResults[code] += 1
		results[i] = code
		if aggResults[code] > aggResults[bestAns] {
			bestAns = code
		}
	}

	log.Printf("Check result for host %s: %v", host, results)
	log.Printf("Aggregated results: %v", aggResults)
	log.Printf("Resource status: %d", bestAns)
	// TODO: How to deal with multiple "right" answers (geo-specific access restriction, etc.)?
}

func init() {
	rand.Seed(time.Now().UnixNano()) // this way of seeding random is probably insecure
}
