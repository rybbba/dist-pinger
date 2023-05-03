package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/rybbba/dist-pinger/client"
	"github.com/rybbba/dist-pinger/reputation"
	"github.com/rybbba/dist-pinger/server"
)

var (
	// client variables
	id = flag.String("id", "", "The external id that must be the same as the external server address of this node") // TODO: remove and replace with cryptography
	//nodeFile = flag.String("file", "nodes.json", "Path to file with nodes information")

	// server variables
	port = flag.Int("port", 50051, "The server port")
)

func main() {
	log.Printf("Running dist-pinger")
	flag.Parse()
	addrs := flag.Args() // list of nodes addresses

	reputationManager := reputation.ReputationManager{}
	reputationManager.InitZeros(addrs)

	pingerServer := server.PingerServer{RepManager: &reputationManager}
	go pingerServer.Serve(*port)

	pingerClient := client.PingerClient{RepManager: &reputationManager}
	pingerClient.SetNodes(addrs)
	pingerClient.SetId(*id)
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
