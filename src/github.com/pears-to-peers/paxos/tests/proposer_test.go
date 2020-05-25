package tests

import (
	"fmt"
	"github.com/pears-to-peers/paxos/paxosnode"
	"github.com/pears-to-peers/paxos/proposer"
	"github.com/pears-to-peers/utils"
	"strconv"
	"testing"
)

func TestNewProposer(t *testing.T) {
	cluster := make([]utils.NodeInfo, 0, 5)
	nodeInfo := new(utils.NodeInfo)
	nodeInfo.NodeID = uint32(6)
	port := 9006
	nodeInfo.HostPort = "localhost:" + strconv.Itoa(port)

	if _, err := proposer.NewProposer(cluster, nodeInfo); err != nil {
		t.Error(err)
	}
}

func TestSendPrepare(t *testing.T) {
	fmt.Println("stat!!!!!!!!!!!")
	cluster := make([]utils.NodeInfo, 0, 5)
	fmt.Println("cluster")
	// populate cluster node info
	for i := 0; i < 5; i++ {
		nodeInfo := new(utils.NodeInfo)
		nodeInfo.NodeID = uint32(i)
		port := 9000 + i
		nodeInfo.HostPort = "localhost:" + strconv.Itoa(port)
		fmt.Println("nodeInfo", nodeInfo)
		cluster = append(cluster, *nodeInfo)
		fmt.Println("cluster", cluster)
	}
	fmt.Println("initialized cluster", cluster)
	// start up the nodes
	nodes := make([]*paxosnode.PaxosNode, 0, 5)
	for i := 0; i < 5; i++ {
		node, _ := paxosnode.NewPaxosNode(cluster[i].HostPort, cluster[i].NodeID, cluster)
		nodes = append(nodes, node)
		fmt.Println("node up", i)
	}
	nodeInfo := new(utils.NodeInfo)
	nodeInfo.NodeID = uint32(6)
	port := 9006
	nodeInfo.HostPort = "localhost:" + strconv.Itoa(port)
	p, _ := proposer.NewProposer(cluster, nodeInfo)
	fmt.Println("proposer", p)
	if err := p.SendPrepare(0, 1234); err != nil {
		t.Error(err)
	}
	for i := 0; i < 5; i++ {
		nodes[i].Close()
	}
}

func TestSendAccept(t *testing.T) {
	value := "I am leader :)"
	fmt.Println("stat!!!!!!!!!!!")
	cluster := make([]utils.NodeInfo, 0, 5)
	fmt.Println("cluster")
	// populate cluster node info
	for i := 0; i < 5; i++ {
		nodeInfo := new(utils.NodeInfo)
		nodeInfo.NodeID = uint32(i)
		port := 9000 + i
		nodeInfo.HostPort = "localhost:" + strconv.Itoa(port)
		fmt.Println("nodeInfo", nodeInfo)
		cluster = append(cluster, *nodeInfo)
		fmt.Println("cluster", cluster)
	}
	fmt.Println("initialized cluster", cluster)
	// start up the nodes
	nodes := make([]*paxosnode.PaxosNode, 0, 5)
	for i := 0; i < 5; i++ {
		node, _ := paxosnode.NewPaxosNode(cluster[i].HostPort, cluster[i].NodeID, cluster)
		nodes = append(nodes, node)
		fmt.Println("node up", i)
	}
	nodeInfo := new(utils.NodeInfo)
	nodeInfo.NodeID = uint32(6)
	port := 9006
	nodeInfo.HostPort = "localhost:" + strconv.Itoa(port)
	p, _ := proposer.NewProposer(cluster, nodeInfo)
	fmt.Println("proposer", p)
	if err := p.SendAccept(value, 0, 1234); err != nil {
		t.Error(err)
	}
	for i := 0; i < 5; i++ {
		nodes[i].Close()
	}
}
