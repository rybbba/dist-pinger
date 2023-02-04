package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"time"

	pb "github.com/rybbba/dist-pinger/pinger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	host = flag.String("host", "example.com", "Host to check")
)

func init() {
	rand.Seed(time.Now().Unix()) // TODO: this way of seeding random is probably unsecure
}

func main() {
	flag.Parse()
	nodes := flag.Args()
	log.Print(nodes)
	addr := nodes[rand.Intn(len(nodes))]
	log.Printf("Using node: %s", addr)

	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewPingerClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r, err := c.CheckHost(ctx, &pb.CheckHostRequest{Host: *host})
	if err != nil {
		log.Fatalf("could not check: %v", err)
	}
	log.Printf("Check result for host %s: %d", *host, r.GetCode())
}
