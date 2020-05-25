package main

import (
	"flag"
	"fmt"
	"github.com/pears-to-peers/cli"
	"strconv"
)

const defaultPort = 8888
const defaultServerPort = 9999

var (
	serverport = flag.Int("serverport", defaultServerPort, "port number central server is running on")
	port       = flag.Int("port", defaultPort, "port number to listen on")
	peers      = flag.String("cluster", "", "file representing cluster")
	quiescing  = flag.String("quiescing", "false", "do you want this node back in the cluster")
	nodeID     = flag.Int("nodeID", 0, "node id of the dead node")
)

func main() {
	flag.Parse()
	if *port == 0 {
		*port = defaultPort
	}
	if *serverport == 0 {
		*serverport = defaultServerPort
	}

	serverHostPort := ":" + strconv.Itoa(*serverport)
	selfHostPort := ":" + strconv.Itoa(*port)

	c, err := cli.NewCli(serverHostPort, selfHostPort)

	if err != nil {
		fmt.Println("error", err)
		return
	}

	if *quiescing == "true" {
		if *peers == "" {
			fmt.Printf("INCORRECT USAGE\n")
			return
		}
		fmt.Println("started quiescing")
		err := c.StartQuiesce(*peers, uint32(*nodeID))
		fmt.Println("end quiescing")
		if err != nil {
			return
		}

		c.Loop()
	}
	//END QUIESCE CASE

	err = c.JoinGame()

	if err != nil {
		fmt.Println("error", err)
		return
	}

	err = c.StartGame()
	if err != nil {
		fmt.Println("error", err)
		return
	}

	c.StartGameLoop()

}
