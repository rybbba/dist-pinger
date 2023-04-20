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

	pingerClient := client.PingerClient{RepManager: &reputationManager, PickCount: 3}
	pingerClient.SetNodes(addrs)
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
