package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"math/rand"
	"time"

	pb "github.com/rybbba/dist-pinger/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Node struct {
	addr string
}

var (
	host     = flag.String("host", "example.com", "Host to check")
	nodeFile = flag.String("file", "nodes.json", "Path to ")
)

func pickN(total int, n int) ([]int, error) {
	if total < n {
		return nil, errors.New("not enough members to pick")
	}
	return rand.Perm(total)[:n], nil
}

func init() {
	rand.Seed(time.Now().UnixNano()) // this way of seeding random is probably insecure
}

const pickCount = 3

func main() {
	flag.Parse()
	addrs := flag.Args()
	nodes := make(map[string]Node)
	for _, addr := range addrs {
		nodes[addr] = Node{addr: addr}
	}
	log.Print(addrs)

	using, err := pickN(len(nodes), pickCount)
	if err != nil {
		log.Fatalf("error while picking nodes: %v", err)
	}
	var results [pickCount]int32
	aggResults := make(map[int32]int)
	var bestAns int32 = 0
	for i, nodeInd := range using {
		node := nodes[addrs[nodeInd]]
		log.Printf("Using node: %v", node)

		conn, err := grpc.Dial(node.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()
		c := pb.NewPingerClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		r, err := c.CheckHost(ctx, &pb.CheckHostRequest{Host: *host})
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

	log.Printf("Check result for host %s: %v", *host, results)
	log.Printf("Aggregated results: %v", aggResults)
	log.Printf("Resource status: %d", bestAns)
	// TODO: How to deal with multiple "right" answers (geo-specific access restriction, etc.)?
}
