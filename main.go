package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/rybbba/dist-pinger/client"
	"github.com/rybbba/dist-pinger/identity"
	"github.com/rybbba/dist-pinger/reputation"
	"github.com/rybbba/dist-pinger/server"
)

var (
	address = flag.String("address", "", "The address (host:port) on which this node will be available for external users")
	keyFile = flag.String("key", "id_rsa", "Path to file with private key")
	//nodeFile = flag.String("file", "nodes.json", "Path to file with nodes information")

	referer = flag.String("ref", "", "Node address to copy initializing ratings from")

	port = flag.Int("port", 50051, "The server port")
)

func main() {
	log.Printf("Running dist-pinger")
	flag.Parse()
	ids := flag.Args() // list of nodes IDs

	key, err := identity.ReadKey(*keyFile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("No private key was provided, generating a new one.")
		} else {
			log.Fatalf("cannot read key file: %v", err)
		}
	}

	selfUser, err := identity.GenUser(*address, key)
	if err != nil {
		log.Fatalf("cannot initialize user keys: %v", err)
	}

	if key == nil { // If there was no key file, we will create it and write our generated key
		err := identity.WriteUserKey(selfUser, *keyFile)
		if err != nil {
			log.Fatalf("cannot write new key: %v", err)
		}
	}

	id := selfUser.Id
	log.Printf("Your ID: %s", id)

	nodeUsers := make([]identity.PublicUser, 0, len(ids))
	for _, id := range ids {
		nodeUser, err := identity.ParseUser(id)
		if err != nil {
			continue
		}
		nodeUsers = append(nodeUsers, nodeUser)
	}

	reputationManager := reputation.ReputationManager{}
	reputationManager.InitNodes(nodeUsers)
	if *referer != "" {
		refUser, err := identity.ParseUser(*referer)
		if err != nil {
			log.Fatalf("error while copying reputations: %v", err)
		}
		err = reputationManager.CopyReputation(selfUser, refUser)
		if err != nil {
			log.Fatalf("error while copying reputations: %v", err)
		}
	}

	pingerServer := server.PingerServer{RepManager: &reputationManager}
	go pingerServer.Serve(*port)

	pingerClient := client.PingerClient{RepManager: &reputationManager}
	pingerClient.SetUser(selfUser)
	for {
		var host string
		n, err := fmt.Scanln(&host)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("Bad input: %v", err)
			continue
		}
		if n != 1 {
			log.Printf("Bad input: no host provided")
			continue
		}
		if host == "r" { // debug output
			fmt.Println(reputationManager.PrintSimpleRep())
			continue
		}

		pingerClient.GetStatus(host)
	}
}
