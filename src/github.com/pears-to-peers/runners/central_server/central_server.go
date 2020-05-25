package main

import (
	"flag"
	"fmt"
	"github.com/pears-to-peers/centralServer"
	"strconv"
)

const defaultPort = 9999

var (
	port = flag.Int("port", defaultPort, "port number to listen on")
)

func main() {
	flag.Parse()

	if *port == 0 {
		*port = defaultPort
	}

	c, err := centralServer.NewCentralServer()
	if err != nil {
		fmt.Println("error", err)
	}
	hostPort := ":" + strconv.Itoa(*port)
	err = c.Start(hostPort)

	if err != nil {
		fmt.Println("could not start the server on port %s", hostPort)
		return
	}

	var quiesce string
	fmt.Println("Do you want to quiesce nodes?")
	fmt.Scanf("%s\n", &quiesce)

	if quiesce == "Y" {
		c.SendQuiesceMessage()
	}
	select {}
}
