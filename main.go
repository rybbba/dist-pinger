package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/rybbba/dist-pinger/client"
	"github.com/rybbba/dist-pinger/server"
	. "github.com/rybbba/dist-pinger/structs"
)

var (
	// client variables
	//nodeFile = flag.String("file", "nodes.json", "Path to file with nodes information")

	// server variables
	port = flag.Int("port", 50051, "The server port")
)

func main() {
	log.Printf("Running dist-pinger")
	flag.Parse()

	pingerServer := server.PingerServer{}
	go pingerServer.Serve(*port)

	addrs := flag.Args() // list of nodes addresses
	nodes := make([]Node, len(addrs))
	for i, addr := range addrs {
		nodes[i] = Node{Address: addr}
	}
	pingerClient := client.PingerClient{PickCount: 3}
	pingerClient.SetNodes(nodes)
	for {
		var host string
		n, err := fmt.Scanln(&host)
		if err != nil {
			log.Printf("Bad input: %v", err)
			continue
		}
		if n != 1 {
			log.Printf("Bad input: no host provided")
			continue
		}

		pingerClient.GetStatus(host)
	}
}
