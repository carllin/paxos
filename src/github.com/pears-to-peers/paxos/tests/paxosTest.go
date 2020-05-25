package main

import (
	"fmt"
	"github.com/pears-to-peers/utils"

	"bufio"
	"github.com/pears-to-peers/paxos"
	"github.com/pears-to-peers/paxos/paxosnode"
	"io"
	"math/rand"
	"os"
	"strconv"
	//"errors"
	"encoding/json"
	"os/exec"
	"strings"
	"time"
)

const (
	NUM_NODES uint32 = 3
)

//TODO: Make sure no nodes get nodeID = 1, returns beginning with nodeID = 1
func generateNodes(numNodes uint32) []utils.NodeInfo {
	var i uint32
	newCluster := make([]utils.NodeInfo, numNodes)
	for i = 1; i <= numNodes; i++ {
		hostport := ":" + strconv.Itoa(int(rand.Int31n(10000)+10000))
		newCluster[i-1] = utils.NodeInfo{HostPort: hostport, NodeID: i}
	}

	return newCluster
}

func remove(Nodes []utils.NodeInfo, index int) []utils.NodeInfo {
	return append(Nodes[:index], Nodes[index+1:]...)
}

func NewPaxosNode() error {
	cluster := make([]utils.NodeInfo, 2)
	_, err := paxosnode.NewPaxosNode(":"+strconv.Itoa(int(rand.Int31n(10000)+10000)), 1, cluster, 0, false)
	if err != nil {
		fmt.Printf("FAIL\n")
		return err
	}
	fmt.Printf("PASS\n")
	return nil
}

func StartRound(Node *paxosnode.PaxosNode, msg string, update chan bool, delay int) {
	time.Sleep(time.Duration(delay) * time.Second)
	Node.StartPaxosRoundFor(msg)
	update <- true
}

func CheckConsistent(filename string, values map[uint32]string, num_values int) bool {
	var entry paxos.LogEntry
	countValues := 0
	file, err := os.OpenFile(filename, os.O_RDWR, 0666)
	if err != nil {
		return false
	}
	Reader := bufio.NewReader(file)

	for {
		mainstring, err := Reader.ReadString('\n')
		if err == io.EOF {
			if countValues == num_values {
				return true
			}

			return false
		}

		if strings.Contains(mainstring, ":true") {
			json.Unmarshal([]byte(mainstring), &entry)

			if values[uint32(entry.SlotNumber)] != entry.Vaccepted {
				return false
			}

			countValues += 1
			continue
		}
	}

	return true
}

func CheckLogConsistent(cluster []utils.NodeInfo, numPlayers int) (bool, error) {
	var entry paxos.LogEntry
	num_committed_values := 0
	committedValues := make(map[uint32]string)
	filename := "/tmp/paxosLog_" + strconv.Itoa(int(cluster[0].NodeID))
	file, err := os.OpenFile(filename, os.O_RDWR, 0666)
	if err != nil {
		return false, err
	}
	Reader := bufio.NewReader(file)

	for {
		mainstring, err := Reader.ReadString('\n')
		if err == io.EOF {
			break
		}

		if strings.Contains(mainstring, ":true") {
			json.Unmarshal([]byte(mainstring), &entry)
			committedValues[uint32(entry.SlotNumber)] = entry.Vaccepted
			num_committed_values += 1
		}
	}

	for i := 1; i < numPlayers; i += 1 {
		otherfilename := "/tmp/paxosLog_" + strconv.Itoa(int(cluster[i].NodeID))
		cmd := exec.Command("diff", filename, otherfilename)
		err := cmd.Run()
		if err != nil {
			return false, nil
		}

	}

	return true, nil

}

func main() {
	NewPaxosNode()
	OnePaxosRound()
	MultiplePaxosRounds(5)
	EveryoneProposes()
	DelayNodes()
	DelayNodes2()
	FailureLimit()
}

func OnePaxosRound() error {
	var i uint32
	var err error
	var ok bool
	Nodes := make(map[uint32]*paxosnode.PaxosNode)
	newCluster := generateNodes(NUM_NODES)
	for i = 0; i < NUM_NODES; i++ {
		Nodes[i], err = paxosnode.NewPaxosNode(newCluster[i].HostPort, newCluster[i].NodeID, newCluster, 0, false)
		if err != nil {
			fmt.Printf("FAIL\n")
			return err
		}
	}

	ok, err = Nodes[0].StartPaxosRoundFor("I am the leader\n")

	if err != nil {
		fmt.Printf("FAIL\n")
		return err
	}

	if ok != true {
		fmt.Printf("FAIL\n")
		return nil
	}

	time.Sleep(time.Duration(3) * time.Second)

	ok, err = CheckLogConsistent(newCluster, int(NUM_NODES))

	if err != nil || !ok {
		fmt.Printf("FAIL\n")
		return err
	}

	fmt.Printf("PASS\n")
	return nil
}

func MultiplePaxosRounds(numRounds uint32) error {
	var i uint32
	var err error
	var ok bool
	Nodes := make(map[uint32]*paxosnode.PaxosNode)
	newCluster := generateNodes(NUM_NODES)
	for i = 0; i < NUM_NODES; i++ {
		Nodes[i], err = paxosnode.NewPaxosNode(newCluster[i].HostPort, newCluster[i].NodeID, newCluster, 0, false)
		if err != nil {
			fmt.Printf("FAIL\n")
			return err
		}
	}

	for i = 0; i < numRounds; i++ {
		ok, err := Nodes[0].StartPaxosRoundFor("I am the leader\n")

		if err != nil {
			fmt.Printf("FAIL\n")
			return err
		}

		if ok != true {
			fmt.Printf("FAIL\n")
			return nil
		}
	}

	time.Sleep(time.Duration(3) * time.Second)

	ok, err = CheckLogConsistent(newCluster, int(NUM_NODES))

	if err != nil || !ok {
		fmt.Printf("FAIL\n")
		return err
	}

	fmt.Printf("PASS\n")
	return nil
}

func EveryoneProposes() error {
	var i uint32
	var err error
	var ok bool
	var count uint32 = 0
	update := make(chan bool, 1)
	Nodes := make(map[uint32]*paxosnode.PaxosNode)
	newCluster := generateNodes(NUM_NODES)
	for i = 0; i < NUM_NODES; i++ {
		Nodes[i], err = paxosnode.NewPaxosNode(newCluster[i].HostPort, newCluster[i].NodeID, newCluster, 0, false)
		if err != nil {
			fmt.Printf("FAIL\n")
			return err
		}
	}

	for i = 0; i < NUM_NODES; i++ {
		go StartRound(Nodes[i], "I "+strconv.Itoa(int(i))+"am the leader\n", update, 0)
	}

	for {
		<-update
		count += 1
		if count == NUM_NODES {
			count = 0
			break
		}
	}

	time.Sleep(time.Duration(5) * time.Second)

	ok, err = CheckLogConsistent(newCluster, int(NUM_NODES))

	if err != nil || !ok {
		fmt.Printf("FAIL\n")
		return err
	}

	fmt.Printf("PASS\n")
	return nil
}

func DelayNodes() error {
	var delay = 3
	var i uint32
	var err error
	var ok bool
	var count uint32 = 0
	update := make(chan bool, 1)
	Nodes := make(map[uint32]*paxosnode.PaxosNode)
	newCluster := generateNodes(NUM_NODES)

	//delay half nodes
	for i = 0; i < NUM_NODES/2; i++ {
		Nodes[i], err = paxosnode.NewPaxosNode(newCluster[i].HostPort, newCluster[i].NodeID, newCluster, delay, false)
		if err != nil {
			fmt.Printf("FAIL\n")
			return err
		}
	}

	for i = NUM_NODES / 2; i < NUM_NODES; i++ {
		Nodes[i], err = paxosnode.NewPaxosNode(newCluster[i].HostPort, newCluster[i].NodeID, newCluster, 0, false)
		if err != nil {
			fmt.Printf("FAIL\n")
			return err
		}
	}

	for i = 0; i < NUM_NODES; i++ {
		go StartRound(Nodes[i], "I am the leader\n", update, 0)
	}

	for {
		<-update
		count += 1
		if count == NUM_NODES {
			count = 0
			break
		}
	}

	time.Sleep(time.Duration(3) * time.Second)

	ok, err = CheckLogConsistent(newCluster, int(NUM_NODES))

	if err != nil || !ok {
		fmt.Printf("FAIL\n")
		return err
	}

	fmt.Printf("PASS\n")
	return nil
}

func DelayNodes2() error {
	var delay = 3
	var delay2 = 2
	var i uint32
	var err error
	var ok bool
	var count uint32 = 0
	update := make(chan bool, 1)
	Nodes := make(map[uint32]*paxosnode.PaxosNode)
	newCluster := generateNodes(NUM_NODES)

	//delay half nodes
	for i = 0; i < NUM_NODES/3; i++ {
		Nodes[i], err = paxosnode.NewPaxosNode(newCluster[i].HostPort, newCluster[i].NodeID, newCluster, delay, false)
		if err != nil {
			fmt.Printf("FAIL\n")
			return err
		}
	}

	for i = NUM_NODES / 3; i < (2*NUM_NODES)/3; i++ {
		Nodes[i], err = paxosnode.NewPaxosNode(newCluster[i].HostPort, newCluster[i].NodeID, newCluster, delay2, false)
		if err != nil {
			fmt.Printf("FAIL\n")
			return err
		}
	}

	for i = (2 * NUM_NODES) / 3; i < NUM_NODES; i++ {
		Nodes[i], err = paxosnode.NewPaxosNode(newCluster[i].HostPort, newCluster[i].NodeID, newCluster, 0, false)
		if err != nil {
			fmt.Printf("FAIL\n")
			return err
		}
	}

	for i = 0; i < NUM_NODES; i++ {
		go StartRound(Nodes[i], "I"+strconv.Itoa(int(i))+"am the leader\n", update, 0)
	}

	for {
		<-update
		count += 1
		if count == NUM_NODES {
			count = 0
			break
		}
	}

	time.Sleep(time.Duration(10) * time.Second)

	ok, err = CheckLogConsistent(newCluster, int(NUM_NODES))

	if err != nil || !ok {
		fmt.Printf("FAIL\n")
		return err
	}

	fmt.Printf("PASS\n")
	return nil
}

//Delays NumToDelay Nodes from receiving message on time
func PartitionNodes(NumToPartition uint32) error {
	var i uint32
	var err error
	var ok bool
	var count uint32 = 0
	update := make(chan bool, 1)
	Nodes := make(map[uint32]*paxosnode.PaxosNode)
	newCluster := generateNodes(NUM_NODES)
	for i = 0; i < NUM_NODES; i++ {
		Nodes[i], err = paxosnode.NewPaxosNode(newCluster[i].HostPort, newCluster[i].NodeID, newCluster, 0, false)
		if err != nil {
			fmt.Printf("FAIL\n")
			return err
		}
	}

	//Partition these nodes
	for i = 0; i < NumToPartition; i++ {
		Nodes[i].Close()
	}

	for i = NumToPartition; i < NUM_NODES; i++ {
		go StartRound(Nodes[i], "I am the leader\n", update, 0)
	}

	for {
		<-update
		count += 1
		if count == NUM_NODES-NumToPartition {
			count = 0
			break
		}
	}

	//Partitioned Nodes Rejoin
	for i = 0; i < NumToPartition; i++ {
		Nodes[i], err = paxosnode.NewPaxosNode(newCluster[i].HostPort, newCluster[i].NodeID, newCluster, 0, false)
		if err != nil {
			fmt.Printf("FAIL\n")
			return err
		}
	}

	//TODO Insert Manual Rejoin of Partitioned Nodes Here...

	time.Sleep(time.Duration(3) * time.Second)

	ok, err = CheckLogConsistent(newCluster, int(NUM_NODES))

	if err != nil || !ok {
		fmt.Printf("FAIL\n")
		return err
	}

	fmt.Printf("PASS\n")
	return nil
}

func FailureLimit() error {
	failLimit := (NUM_NODES - 1) / 2
	var i uint32
	var err error
	var ok bool
	var count uint32 = 0
	update := make(chan bool, 1)
	Nodes := make(map[uint32]*paxosnode.PaxosNode)
	newCluster := generateNodes(NUM_NODES)
	for i = 0; i < NUM_NODES; i++ {
		Nodes[i], err = paxosnode.NewPaxosNode(newCluster[i].HostPort, newCluster[i].NodeID, newCluster, 0, false)
		if err != nil {
			fmt.Printf("FAIL\n")
			return err
		}
	}

	//Partition these nodes
	for i = 0; i < failLimit; i++ {
		Nodes[i].Close()
	}

	//send multiple rounds of messages
	for j := 0; j < 5; j++ {
		for i = failLimit; i < NUM_NODES; i++ {
			go StartRound(Nodes[i], "I am the leader\n", update, 0)
		}

		for {
			<-update
			count += 1
			if count == NUM_NODES-failLimit {
				count = 0
				break
			}
		}
	}

	time.Sleep(time.Duration(10) * time.Second)

	ok, err = CheckLogConsistent(newCluster[failLimit:], int(NUM_NODES-failLimit))

	if err != nil || !ok {
		fmt.Printf("FAIL\n")
		return err
	}

	fmt.Printf("PASS\n")
	return nil
}
