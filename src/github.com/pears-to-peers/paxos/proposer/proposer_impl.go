// sends out prepare, accept, commit messages to the other cluster members
package proposer

import (
	"encoding/json"
	"fmt"
	"github.com/pears-to-peers/paxos"
	"io/ioutil"
	"log"
	"net"
	"os"
)

const (
	timeout = 10000
)

type proposer struct {
	commonInfo      *paxos.Slot
	numberOfAccepts int // do we need separate accept counter for each stage?
	Accepted_IDs    map[uint32]bool
}

var LOGE = log.New(os.Stderr, "ERROR ", log.Lmicroseconds|log.Lshortfile)
var LOGV = log.New(ioutil.Discard, "VERBOSE ", log.Lmicroseconds|log.Lshortfile)

func NewProposer(commonInfo *paxos.Slot) (Proposer, error) {
	proposer := new(proposer)
	proposer.commonInfo = commonInfo
	proposer.Accepted_IDs = make(map[uint32]bool)
	proposer.numberOfAccepts = 0
	return proposer, nil
}

func (p *proposer) SendPrepare(slotNumber int, proposalNumber uint32) error {
	message := new(paxos.PaxosMessage)
	message.Type = paxos.Prepare
	message.SlotNumber = slotNumber
	message.ProposalNumber = proposalNumber
	message.ProposerID = p.commonInfo.NodeInfo.NodeID

	// make tcp connection to all other nodes in the cluster we are aware of
	// send them the prepare message
	p.numberOfAccepts = 1 // our accept
	p.Accepted_IDs = make(map[uint32]bool)
	numberOfFailedSends := 0

	LOGV.Println("about to dial")
	for i := 0; i < len(p.commonInfo.ClusterMembers); i++ {
		if p.commonInfo.ClusterMembers[i].NodeID == p.commonInfo.NodeInfo.NodeID {
			continue
		}
		conn, err := net.Dial("tcp", p.commonInfo.ClusterMembers[i].HostPort)
		if err != nil {
			numberOfFailedSends++
			continue
		}
		buf, err := json.Marshal(message)
		if err != nil {
			numberOfFailedSends++
			continue
		}
		LOGV.Printf("sending message to node %d : %s", p.commonInfo.ClusterMembers[i].NodeID, string(buf))
		_, err = conn.Write(buf)
		if err != nil {
			numberOfFailedSends++
		}
	}

	LOGV.Println("done dial")
	if numberOfFailedSends == len(p.commonInfo.ClusterMembers) {
		return fmt.Errorf("Could not send prepare to any node")
	}
	return nil
}

func (p *proposer) SendAccept(value string, slotNumber int, proposalNumber uint32) error {
	// Write out to log here to indicate self-accept
	// optimization to not send it to urself :)

	message := new(paxos.PaxosMessage)
	message.Type = paxos.Accept
	message.SlotNumber = slotNumber
	message.Value = value
	message.ProposalNumber = proposalNumber
	message.ProposerID = p.commonInfo.NodeInfo.NodeID

	// make tcp connection to all other nodes in the cluster we are aware of
	// send them the accept message
	numberOfFailedSends := 0

	p.numberOfAccepts = 1
	p.Accepted_IDs = make(map[uint32]bool)

	LOGV.Println("about to dial")
	for i := 0; i < len(p.commonInfo.ClusterMembers); i++ {
		if p.commonInfo.ClusterMembers[i].NodeID == p.commonInfo.NodeInfo.NodeID {
			continue
		}
		conn, err := net.Dial("tcp", p.commonInfo.ClusterMembers[i].HostPort)
		if err != nil {
			numberOfFailedSends++ // we cudnt dial in to some node, so ignore
			continue
		}
		buf, err := json.Marshal(message)
		if err != nil {
			numberOfFailedSends++
			continue
		}

		LOGV.Printf("sending message to node %d : %s", p.commonInfo.ClusterMembers[i].NodeID, string(buf))
		_, err = conn.Write(buf)
		if err != nil {
			numberOfFailedSends++
		}

	}

	LOGV.Println("done dial")
	if numberOfFailedSends == len(p.commonInfo.ClusterMembers) {
		return fmt.Errorf("Could not send prepare to any node")
	}
	return nil
}

func (p *proposer) SendCommit(value string, slotNumber int, proposalNumber uint32) error {
	message := new(paxos.PaxosMessage)
	message.Type = paxos.Commit
	message.SlotNumber = slotNumber
	message.Value = value
	message.ProposalNumber = proposalNumber
	message.ProposerID = p.commonInfo.NodeInfo.NodeID

	// make tcp connection to all other nodes in the cluster we are aware of
	// send them the prepare message
	numberOfFailedSends := 0
	LOGV.Println("about to dial")
	for i := 0; i < len(p.commonInfo.ClusterMembers); i++ {
		if p.commonInfo.ClusterMembers[i].NodeID == p.commonInfo.NodeInfo.NodeID {
			continue
		}
		conn, err := net.Dial("tcp", p.commonInfo.ClusterMembers[i].HostPort)
		if err != nil {
			numberOfFailedSends++
			continue
		}
		buf, err := json.Marshal(message)
		if err != nil {
			numberOfFailedSends++
			continue
		}
		LOGV.Printf("sending message to node %d : %s", p.commonInfo.ClusterMembers[i].NodeID, string(buf))
		_, err = conn.Write(buf)
		if err != nil {
			numberOfFailedSends++
		}
	}
	LOGV.Println("done dial")
	if numberOfFailedSends == len(p.commonInfo.ClusterMembers) {
		return fmt.Errorf("Could not send prepare to any node")
	}
	return nil
}
