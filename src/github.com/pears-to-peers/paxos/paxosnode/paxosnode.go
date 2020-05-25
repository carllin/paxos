// The core of the paxos implementation is here.
// It has a slot manager goroutine. Slots are in-memory replica of the paxos log.
// startPaxosRound is used for adding a value to the paxos log

package paxosnode

import (
	"encoding/json"
	"fmt"
	"github.com/pears-to-peers/paxos"
	"github.com/pears-to-peers/paxos/acceptor"
	"github.com/pears-to-peers/paxos/proposer"
	"github.com/pears-to-peers/utils"
	"net"
	// "errors"
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"
)

const (
	timeout          = 10000
	FAILURE_LIMIT    = 5
	MAX_SLOTS        = 100
	MAX_SLOT_BYTES   = 200
	startRound       = "StartPaxosRound"
	accept           = "accept"
	prepare          = "prepare"
	writeToSlot      = "writeToSlot"
	acceptReject     = "acceptReject"
	prepareReject    = "prepareReject"
	acceptOK         = "acceptOK"
	prepareOK        = "prepareOK"
	commit           = "commit"
	startNilOPRound  = "startNilOPRound"
	MAX_CLUSTER_SIZE = 8
)

// encapsulates the operation we want to perform on the slots
type Operation struct {
	Name           string // operation to be performed
	SlotNumber     int
	ProposalNumber uint32
	Value          string
	ProposerID     uint32
	ReplyChan      chan SlotInfo // the reply back channel
	PaxosMessage   paxos.PaxosMessage
	Committed      bool
}

type PaxosNode struct {
	proposer          proposer.Proposer
	acceptor          acceptor.Acceptor
	ln                net.Listener
	shutDown          chan bool
	connections       []net.Conn
	logFileName       string
	highestSlotNumber int // to keep track of the highest slot we are contending for, because
	// current paxos round could be for an old slot
	reader           *bufio.Reader
	slots            []*paxos.Slot
	requestChan      chan *Operation
	hostPort         string
	nodeID           uint32
	clusterMembers   []utils.NodeInfo
	MessageAvailable chan bool
	readSlot         int
	QuiesceStarted   bool
	isProposing      bool
	QuiesceWait      chan bool
	NilOPWait        chan bool
	delay            int
}

type SlotInfo struct {
	SlotNumber     int
	ProposalNumber uint32
	ValueAccepted  string
}

var LOGE = log.New(ioutil.Discard, "ERROR ", log.Lmicroseconds|log.Lshortfile)
var LOGV = log.New(ioutil.Discard, "VERBOSE ", log.Lmicroseconds|log.Lshortfile)

func NewSlot(nodeID uint32, clusterMembers []utils.NodeInfo, hostPort string) *paxos.Slot {
	newslot := new(paxos.Slot)
	newslot.ClusterMembers = clusterMembers
	newslot.Naccepted = 0
	newslot.Vaccepted = ""
	newslot.NumberOfPrepareOK = 0
	newslot.NumberOfAcceptOK = 0
	newslot.AcceptOKIDs = make(map[uint32]bool)
	newslot.PrepareOKIDs = make(map[uint32]bool)
	newslot.MajorityReceivedForPrepare = make(chan bool, len(clusterMembers)) // we dont want to block if no one is listening
	newslot.MajorityReceivedForAccept = make(chan bool, len(clusterMembers))  // we dont want to block if no one is listening
	newslot.ProposalRejected = make(chan bool, len(clusterMembers))           // we dont want to block if no one is listening
	node := new(utils.NodeInfo)
	node.HostPort = hostPort
	node.NodeID = nodeID
	newslot.NodeInfo = node
	newslot.CurrentSlotNumber = 0
	newslot.Nhighest = 0
	newslot.Committed = false

	return newslot
}

func NewPaxosNode(hostPort string, nodeID uint32, clusterMembers []utils.NodeInfo, delay int, quiescing bool) (*PaxosNode, error) {
	pNode := new(PaxosNode)
	pNode.nodeID = nodeID
	pNode.clusterMembers = clusterMembers
	pNode.hostPort = hostPort
	commonInfo := NewSlot(nodeID, clusterMembers, hostPort)
	pNode.proposer, _ = proposer.NewProposer(commonInfo)
	pNode.acceptor = acceptor.NewAcceptor(commonInfo)
	pNode.shutDown = make(chan bool)
	pNode.connections = make([]net.Conn, 0, len(clusterMembers))
	pNode.highestSlotNumber = -1

	pNode.slots = make([]*paxos.Slot, 0, MAX_SLOTS)

	pNode.requestChan = make(chan *Operation)
	pNode.MessageAvailable = make(chan bool, 2*MAX_SLOTS)
	pNode.readSlot = 0
	pNode.isProposing = false
	pNode.QuiesceStarted = false
	pNode.QuiesceWait = make(chan bool)
	pNode.NilOPWait = make(chan bool)
	pNode.delay = delay

	for i := 0; i < MAX_SLOTS; i++ {
		slot := NewSlot(pNode.nodeID, pNode.clusterMembers, pNode.hostPort)
		pNode.slots = append(pNode.slots, slot)
	}

	ln, err := net.Listen("tcp", hostPort)
	pNode.ln = ln
	if err != nil {
		// handle error
		LOGE.Println("listen error", err)
		return nil, err
	}
	fileName := fmt.Sprintf("/tmp/paxosLog_%d", nodeID)
	pNode.logFileName = fileName
	if !quiescing {
		var f *os.File
		f, err = os.Create(fileName)

		if err != nil {
			LOGE.Println("Could not create paxos log")
			return nil, err
		}
		f.Close()
		pNode.writeOutMaxSlots()
	}
	readLog, _ := os.OpenFile(fileName, os.O_RDWR, 0666)
	pNode.reader = bufio.NewReader(readLog)

	go pNode.manageSlots()
	go pNode.acceptConnections()

	return pNode, nil
}
func (n *PaxosNode) writeOutMaxSlots() {
	writeLog, _ := os.OpenFile(n.logFileName, os.O_RDWR, 0666)
	for i := 0; i < MAX_SLOTS; i++ {
		writeLog.Seek(int64(i*MAX_SLOT_BYTES), 0)
		writeLog.WriteString("\n")
	}
	writeLog.Sync()
	writeLog.Close()
}

// main method used to append to log

func (n *PaxosNode) StartPaxosRoundFor(value string) (bool, error) {
	if n.QuiesceStarted == true {
		return false, fmt.Errorf("QuiesceStarted!!!")
	}
	n.isProposing = true
	operation := n.getOperationObjectForStartRound(startRound, value)
	n.requestChan <- operation
	reply := <-operation.ReplyChan
	slotNumber := reply.SlotNumber
	proposalNumber := reply.ProposalNumber

	// var num_failures uint32 = 0
	err := n.proposer.SendPrepare(slotNumber, proposalNumber)
	if err != nil {
		n.isProposing = false
		if n.QuiesceStarted {
			n.QuiesceWait <- false
		}

		return false, err
	}
	// wait until timeout for receiving majority prepare ok
	// use a select with a channel waiting and the other being a timeout
	// if timeout, try sendPrepare with a higher proposalNumber
	// else we have majority so sendAccept
	var acceptedValue string
	select {
	// case <-time.After(time.Duration(timeout) * time.Millisecond):
	// 	fmt.Println("oops timed out, we'll have to retry")
	//     num_failures++
	//     if num_failures == FAILURE_LIMIT{
	//         return nil, errors.New("PAXOS ROUND TIMED OUT\n")
	//     }
	//     time.Sleep(time.Duration(timeout*num_failures)*time.Millisecond)
	//     n.proposer.SendPrepare(n.commonInfo.CurrentSlotNumber, proposalNumber)
	case <-n.slots[slotNumber].ProposalRejected:
		//fmt.Printf("our proposal has been rejected! waiting for somebody else to be leader\n")
		n.isProposing = false
		if n.QuiesceStarted {
			n.QuiesceWait <- false
		}
		return false, nil
	case <-n.slots[slotNumber].MajorityReceivedForPrepare:
		operation := n.getOperationObjectForSlotNumber(writeToSlot, slotNumber, false)
		n.requestChan <- operation
		reply := <-operation.ReplyChan
		acceptedValue = reply.ValueAccepted
		time.Sleep(time.Duration(n.delay) * time.Second)
		n.proposer.SendAccept(acceptedValue, slotNumber, proposalNumber)
	}

	select {
	// case <-time.After(time.Duration(timeout) * time.Millisecond):
	// 	fmt.Println("oops timed out, we'll have to retry")
	// 	num_failures++
	// 	if num_failures == FAILURE_LIMIT{
	// 	    return nil, errors.New("PAXOS ROUND TIMED OUT\n")
	// 	}
	//     time.Sleep(time.Duration(timeout*num_failures)*time.Millisecond)
	//     n.proposer.SendPrepare(n.commonInfo.CurrentSlotNumber, proposalNumber)

	case <-n.slots[slotNumber].ProposalRejected:
		LOGE.Println("our proposal has been rejected! waiting for somebody else to be leader")
		n.isProposing = false
		if n.QuiesceStarted {
			n.QuiesceWait <- false
		}
		return false, nil
	case <-n.slots[slotNumber].MajorityReceivedForAccept:
		operation := n.getOperationObjectForSlotNumber(writeToSlot, slotNumber, true)
		n.requestChan <- operation
		_ = <-operation.ReplyChan
		// update the log value to committed = true
		time.Sleep(time.Duration(n.delay) * time.Second)
		n.proposer.SendCommit(acceptedValue, slotNumber, proposalNumber)
	}
	n.slots[slotNumber].Committed = true
	n.isProposing = false
	if n.QuiesceStarted {
		n.QuiesceWait <- true
	}
	return true, nil
}

// method used to return committed messages from the paxos log in order
func (n *PaxosNode) NextMessage() (*paxos.LogEntry, error) {
	// read the message from the log for the current slot nnumber and return its value to the player
	<-n.MessageAvailable
	readLog, _ := os.OpenFile(n.logFileName, os.O_RDWR, 0666)
	readLog.Seek(int64(n.readSlot*MAX_SLOT_BYTES), 0)
	var buf []byte
	var marshalledLogEntry []byte
	buf = make([]byte, 1, 1)
	for {
		_, err := readLog.Read(buf[0:1])
		if err == io.EOF {
			// do something here
			// fmt.Println("EOF in log", string(marshalledLogEntry))
			return nil, fmt.Errorf("File EOF occured")
		} else if err != nil {
			LOGE.Println("some other  error", err)
			return nil, err // if you return error
		} else {
			marshalledLogEntry = append(marshalledLogEntry, buf[0])
			if buf[0] == '\n' {
				if len(marshalledLogEntry) == 1 {
					return nil, nil
				}
				break
			}
		}
	}

	var entry *paxos.LogEntry
	err := json.Unmarshal(marshalledLogEntry, &entry)
	if err == nil {
		if entry.Committed == false {
			return nil, nil
		}
	}

	n.readSlot++
	return entry, nil
}

func (n *PaxosNode) generateProposalNumber(slotNumber int) uint32 {
	if len(n.slots) == 0 || len(n.slots) < slotNumber {
		return n.nodeID
	}
	slot := n.slots[slotNumber]
	if slot.Nhighest == 0 { // no proposals seen yet
		return n.nodeID
	} else {
		return slot.Nhighest + 1 // choose number one higher than the highest we have seen
	}

}

// this is used in quiesce period for filling uncommitted slots
func (n *PaxosNode) StartPaxosRoundForSlot(slotNumber int, value string) (bool, error) {
	operation := n.getOperationObjectForStartNilOpRound(startNilOPRound, value, slotNumber)
	n.requestChan <- operation
	reply := <-operation.ReplyChan
	proposalNumber := reply.ProposalNumber

	// var num_failures uint32 = 0
	err := n.proposer.SendPrepare(slotNumber, proposalNumber)
	if err != nil {
		n.NilOPWait <- false
		return false, err
	}
	// wait until timeout for receiving majority prepare ok
	// use a select with a channel waiting and the other being a timeout
	// if timeout, try sendPrepare with a higher proposalNumber
	// else we have majority so sendAccept
	var acceptedValue string
	select {
	// case <-time.After(time.Duration(timeout) * time.Millisecond):
	// 	fmt.Println("oops timed out, we'll have to retry")
	//     num_failures++
	//     if num_failures == FAILURE_LIMIT{
	//         return nil, errors.New("PAXOS ROUND TIMED OUT\n")
	//     }
	//     time.Sleep(time.Duration(timeout*num_failures)*time.Millisecond)
	//     n.proposer.SendPrepare(n.commonInfo.CurrentSlotNumber, proposalNumber)
	case <-n.slots[slotNumber].ProposalRejected:
		LOGE.Println("our proposal has been rejected! waiting for somebody else to be leader")
		n.NilOPWait <- false
		return false, nil
	case <-n.slots[slotNumber].MajorityReceivedForPrepare:
		operation := n.getOperationObjectForSlotNumber(writeToSlot, slotNumber, false)
		n.requestChan <- operation
		reply := <-operation.ReplyChan
		acceptedValue = reply.ValueAccepted
		n.proposer.SendAccept(acceptedValue, slotNumber, proposalNumber)
	}

	select {
	// case <-time.After(time.Duration(timeout) * time.Millisecond):
	// 	fmt.Println("oops timed out, we'll have to retry")
	// 	num_failures++
	// 	if num_failures == FAILURE_LIMIT{
	// 	    return nil, errors.New("PAXOS ROUND TIMED OUT\n")
	// 	}
	//     time.Sleep(time.Duration(timeout*num_failures)*time.Millisecond)
	//     n.proposer.SendPrepare(n.commonInfo.CurrentSlotNumber, proposalNumber)

	case <-n.slots[slotNumber].ProposalRejected:
		LOGE.Println("our proposal has been rejected! waiting for somebody else to be leader")
		n.NilOPWait <- false
		return false, nil
	case <-n.slots[slotNumber].MajorityReceivedForAccept:
		operation := n.getOperationObjectForSlotNumber(writeToSlot, slotNumber, true)
		n.requestChan <- operation
		_ = <-operation.ReplyChan
		// update the log value to committed = true
		n.proposer.SendCommit(acceptedValue, slotNumber, proposalNumber)
	}
	n.NilOPWait <- true
	return true, nil
}

func (n *PaxosNode) getOperationObjectForSlotNumber(operation string, slotNumber int, committed bool) *Operation {
	op := new(Operation)
	op.Name = operation
	op.SlotNumber = slotNumber
	op.ReplyChan = make(chan SlotInfo)
	op.Committed = committed
	return op
}

func (n *PaxosNode) getOperationObjectForStartRound(operation string, value string) *Operation {
	op := new(Operation)
	op.Name = operation
	op.ReplyChan = make(chan SlotInfo)
	op.Value = value
	return op
}

func (n *PaxosNode) getOperationObjectForStartNilOpRound(operation string, value string, slotNumber int) *Operation {
	op := new(Operation)
	op.Name = operation
	op.ReplyChan = make(chan SlotInfo)
	op.Value = value
	op.SlotNumber = slotNumber
	return op
}

func (n *PaxosNode) getOperationObjectForProposal(operation string, slotNumber int, proposalNumber uint32) *Operation {
	op := new(Operation)
	op.Name = operation
	op.ReplyChan = make(chan SlotInfo)
	op.SlotNumber = slotNumber
	op.ProposalNumber = proposalNumber
	return op
}

func (n *PaxosNode) getOperationObjectForValues(operation string, slotNumber int, proposalNumber uint32, proposerId uint32, value string) *Operation {
	op := new(Operation)
	op.Name = operation
	op.ReplyChan = make(chan SlotInfo)
	op.SlotNumber = slotNumber
	op.ProposalNumber = proposalNumber
	op.ProposerID = proposerId
	op.Value = value
	return op
}

func (n *PaxosNode) getOperationObjectForMessage(operation string, message paxos.PaxosMessage) *Operation {
	op := new(Operation)
	op.Name = operation
	op.PaxosMessage = message
	return op
}

func (n *PaxosNode) acceptConnections() {
	// fmt.Println("ready to listen to other players")
	for {
		conn, err := n.ln.Accept()
		n.connections = append(n.connections, conn)
		if err != nil {
			// handle error
			LOGE.Println("error in accept", err)
			return // error in our listener
		} else {
			LOGV.Println("receieved a connection")
			go n.handleConnections(conn)
		}
	}
}

// interprets network messages
func (n *PaxosNode) handleConnections(conn net.Conn) {

	/*
		Read from conn
		switch on the message type
		if message is PrepareOk
			call proposer.ReceivePrepareOK
			if the call returns true(it means our proposer has majority for this slot yay!!!)
				then put something on that channel
		if message is AcceptOK
			call proposer.ReceiveAcceptOK
		if message is Prepare
			call acceptor.ReceivePrepare
		if message is Accept
			call acceptor.ReceiveAccept
		if message is Commit
			call acceptor.ReceiveCommit
	*/

	var buf [1024]byte
	for {
		rlen, err := conn.Read(buf[:])
		if err != nil { // error in reading from connection
			LOGV.Println("err in read", err)
			return // kill the gorouting
		} else {
			var msg paxos.PaxosMessage
			_ = json.Unmarshal(buf[0:rlen], &msg)
			LOGV.Println("Recieved msg", msg)
			switch msgType := msg.Type; {
			case msgType == paxos.Prepare:
				LOGV.Println("Prepare")
				operation := n.getOperationObjectForMessage(prepare, msg)
				n.requestChan <- operation
			case msgType == paxos.PrepareOK:
				LOGV.Println("PrepareOK")
				operation := n.getOperationObjectForMessage(prepareOK, msg)
				n.requestChan <- operation
			case msgType == paxos.PrepareReject:
				operation := n.getOperationObjectForSlotNumber(prepareReject, msg.SlotNumber, false)
				n.requestChan <- operation
			case msgType == paxos.Accept:
				// TODO: worry about log consistency
				LOGV.Println("Accept")
				operation := n.getOperationObjectForMessage(accept, msg)
				n.requestChan <- operation
			case msgType == paxos.AcceptOK:
				LOGV.Println("AcceptOK")
				operation := n.getOperationObjectForMessage(acceptOK, msg)
				n.requestChan <- operation
			case msgType == paxos.AcceptReject:
				operation := n.getOperationObjectForSlotNumber(acceptReject, msg.SlotNumber, false)
				n.requestChan <- operation
			case msgType == paxos.Commit:
				LOGV.Println("Commit")
				operation := n.getOperationObjectForMessage(commit, msg)
				n.requestChan <- operation
				n.acceptor.ReceiveCommit()
			case msgType == paxos.Quiesce:
				deadNodeId := msg.ProposerID
				n.StartQuiesce(deadNodeId)
			}

		}
	}
}

func (n *PaxosNode) Close() {
	for i := 0; i < len(n.connections); i++ {
		n.connections[i].Close()
	}
	n.ln.Close()
}

// write a slot value in the paxos log
func (n *PaxosNode) writeAtSlot(slotNumber int, buf []byte) error {
	writeLog, _ := os.OpenFile(n.logFileName, os.O_RDWR, 0666)
	offset := int64(slotNumber * MAX_SLOT_BYTES)
	writeLog.Seek(offset, 0) // from origin of file go to current offset
	nbytes, err := writeLog.WriteString(string(buf) + "\n")
	if err != nil {
		LOGE.Printf("Error in writing to log")
		return err
	}
	LOGV.Printf("wrote %d bytes\n", nbytes)
	writeLog.Sync()
	writeLog.Close()

	n.MessageAvailable <- true

	return nil
}

func (n *PaxosNode) resetSlotValues(slotNumber int) {
	slot := n.slots[slotNumber]
	slot.CurrentSlotNumber = slotNumber // received a prepare for this slot hence setting current slot number
	slot.Naccepted = 0
	slot.Vaccepted = ""
}

// slot manager
func (n *PaxosNode) manageSlots() {
	for {
		select {
		case item := <-n.requestChan: // monitor the request channel
			switch operation := item.Name; {
			case operation == startRound: // delete connection
				n.highestSlotNumber++
				value := item.Value
				proposalNumber := n.generateProposalNumber(n.highestSlotNumber)
				slot := n.slots[n.highestSlotNumber]
				slot.Nhighest = proposalNumber
				slot.MyValue = value
				slot.CurrentSlotNumber = n.highestSlotNumber // we are starting the round so set the current round number to this
				slot.NumberOfPrepareOK = 1
				slotInfo := new(SlotInfo)
				slotInfo.ProposalNumber = proposalNumber
				slotInfo.SlotNumber = slot.CurrentSlotNumber
				item.ReplyChan <- *slotInfo
			case operation == writeToSlot: //new connection
				slotNumber := item.SlotNumber
				committed := item.Committed
				slot := n.slots[slotNumber]
				// now we write out to log, we are accepting our own value
				LOGV.Println("Vaccepted", slot.Vaccepted)
				entry := new(paxos.LogEntry)
				entry.SlotNumber = slotNumber
				entry.ProposerID = slot.NodeInfo.NodeID
				entry.Vaccepted = slot.Vaccepted
				entry.NumberOfPlayers = len(slot.ClusterMembers)
				entry.Committed = committed

				buf, err := json.Marshal(entry)
				err = n.writeAtSlot(slotNumber, buf)
				if err != nil {

				} else {
					slotInfo := new(SlotInfo)
					slotInfo.ValueAccepted = slot.Vaccepted
					item.ReplyChan <- *slotInfo
				}
			case operation == prepare:
				msg := item.PaxosMessage
				slotNumber := msg.SlotNumber
				proposalNumber := msg.ProposalNumber
				slot := n.slots[slotNumber]
				n.resetSlotValues(slotNumber)
				if slotNumber > n.highestSlotNumber {
					n.highestSlotNumber = slot.CurrentSlotNumber
				}
				n.ReceivePrepare(&msg, slot)
				if slotNumber == slot.CurrentSlotNumber && proposalNumber > slot.Nhighest { // this prepare is for the same slot and with a higher proposal number
					slot.ProposalRejected <- true
				}
			case operation == accept:
				msg := item.PaxosMessage
				slotNumber := msg.SlotNumber
				proposalNumber := msg.ProposalNumber
				//proposerId := msg.ProposerID
				//value := msg.Value
				slot := n.slots[slotNumber]
				if slot.CurrentSlotNumber != slotNumber {
					n.resetSlotValues(slotNumber)
					LOGV.Println("reset offset in accept")
				}

				n.ReceiveAccept(&msg, slot) //send to myself here

				if slotNumber == slot.CurrentSlotNumber && proposalNumber > slot.Nhighest { // this prepare is for the same slot and with a higher proposal number
					slot.ProposalRejected <- true
				}
			case operation == acceptReject:
				slotNumber := item.SlotNumber
				slot := n.slots[slotNumber]
				if slotNumber == slot.CurrentSlotNumber { // the reject is for the same slot
					slot.ProposalRejected <- true
				}

			case operation == prepareReject:
				slotNumber := item.SlotNumber
				slot := n.slots[slotNumber]
				if slotNumber == slot.CurrentSlotNumber { // the reject is for the same slot
					slot.ProposalRejected <- true
				}
			case operation == acceptOK:
				msg := item.PaxosMessage
				slotNumber := msg.SlotNumber
				slot := n.slots[slotNumber]
				if n.ReceiveAcceptOK(&msg, slot) {
					slot.MajorityReceivedForAccept <- true
				}
			case operation == prepareOK:
				msg := item.PaxosMessage
				slotNumber := msg.SlotNumber
				slot := n.slots[slotNumber]
				if n.ReceivePrepareOK(&msg, slot) { // we received majority
					// put on the waiting channel
					slot.NumberOfAcceptOK = 1
					slot.MajorityReceivedForPrepare <- true
				}
			case operation == commit:
				msg := item.PaxosMessage
				slotNumber := msg.SlotNumber
				proposerId := msg.ProposerID
				value := msg.Value
				slot := n.slots[slotNumber]
				if slot.CurrentSlotNumber != slotNumber {
					n.resetSlotValues(slotNumber)
					LOGV.Println("reset offset in accept")
				}
				slot.Committed = true
				//TODO: there maybe old paxos rounds going on , so we should probably not always blindly write
				entry := new(paxos.LogEntry)
				entry.SlotNumber = slot.CurrentSlotNumber
				entry.ProposerID = proposerId
				entry.Vaccepted = value
				entry.NumberOfPlayers = len(slot.ClusterMembers)
				entry.Committed = true

				buf, _ := json.Marshal(entry)
				n.writeAtSlot(slotNumber, buf)
			case operation == startNilOPRound:
				value := item.Value
				slotNumber := item.SlotNumber
				proposalNumber := n.generateProposalNumber(slotNumber)
				slot := n.slots[slotNumber]
				slot.Nhighest = proposalNumber
				slot.MyValue = value
				slot.CurrentSlotNumber = slotNumber // we are starting the round so set the current round number to this
				slot.NumberOfPrepareOK = 1
				slotInfo := new(SlotInfo)
				slotInfo.ProposalNumber = proposalNumber
				slotInfo.SlotNumber = slot.CurrentSlotNumber
				item.ReplyChan <- *slotInfo
			}
		}
	}
}

// steps to be done when a prepare ok message is received
func (p *PaxosNode) ReceivePrepareOK(OKMsg *paxos.PaxosMessage, slot *paxos.Slot) bool {
	if OKMsg.ProposalNumber != slot.Nhighest {
		return false
	}

	if _, ok := slot.PrepareOKIDs[OKMsg.ProposerID]; ok {
		return false
	}

	slot.PrepareOKIDs[OKMsg.ProposerID] = true

	slot.NumberOfPrepareOK++

	if OKMsg.Naccepted > slot.Naccepted {
		slot.Naccepted = OKMsg.Naccepted
	}
	LOGV.Println(OKMsg.Vaccepted + "Vaccepted" + slot.MyValue)
	if OKMsg.Vaccepted != "" {
		slot.Vaccepted = OKMsg.Vaccepted
		LOGV.Println("set my Vaccepted to", slot.Vaccepted)
	} else {
		slot.Vaccepted = slot.MyValue
		LOGV.Println("set my Vaccepted to", slot.Vaccepted)
	}
	if slot.NumberOfPrepareOK >= ((len(slot.ClusterMembers) + 1) / 2) { //TODO: add one to account for odd yay!!! majority
		return true
	}

	return false
}

// steps to be done when a accept ok message is received
func (p *PaxosNode) ReceiveAcceptOK(OKMsg *paxos.PaxosMessage, slot *paxos.Slot) bool {
	if slot.Nhighest != OKMsg.ProposalNumber {
		return false
	}

	if _, ok := slot.AcceptOKIDs[OKMsg.ProposerID]; ok {
		return false
	}

	slot.AcceptOKIDs[OKMsg.ProposerID] = true
	slot.NumberOfAcceptOK++
	if slot.NumberOfAcceptOK >= ((len(slot.ClusterMembers) + 1) / 2) { // yay!!! majority
		return true
	}
	return false
}

// steps to be done when a prepare message is received
func (a *PaxosNode) ReceivePrepare(PrepMsg *paxos.PaxosMessage, slot *paxos.Slot) error {
	var newMsg paxos.PaxosMessage

	if PrepMsg.ProposalNumber > slot.Nhighest {
		LOGV.Println("higher than we have seen", slot.Vaccepted)
		slot.Nhighest = PrepMsg.ProposalNumber

		newMsg = paxos.PaxosMessage{Type: paxos.PrepareOK, SlotNumber: PrepMsg.SlotNumber, Vaccepted: slot.Vaccepted, ProposalNumber: PrepMsg.ProposalNumber, ProposerID: slot.NodeInfo.NodeID}
	} else {
		LOGV.Println("lower than we have seen")
		newMsg = paxos.PaxosMessage{Type: paxos.PrepareReject, SlotNumber: PrepMsg.SlotNumber, Vaccepted: slot.Vaccepted, Nhighest: slot.Nhighest, ProposalNumber: PrepMsg.ProposalNumber, ProposerID: slot.NodeInfo.NodeID}
	}
	buf, err := json.Marshal(newMsg)
	hostPort := a.findClusterMemberHostPort(PrepMsg.ProposerID, slot)
	if hostPort == "" {
		return fmt.Errorf("No such cluster member?")
	}
	conn, err := net.Dial("tcp", hostPort)
	if err != nil {
		return err
	}
	_, err = conn.Write(buf) // send reply to the proposer
	if err != nil {
		LOGE.Println("could not write")
		return err
	}
	return nil
}

func (a *PaxosNode) findClusterMemberHostPort(ProposerID uint32, slot *paxos.Slot) string {
	for i := 0; i < len(slot.ClusterMembers); i++ {
		if slot.ClusterMembers[i].NodeID == ProposerID {
			return slot.ClusterMembers[i].HostPort
		}
	}
	return ""
}

// steps to be done when a accept message is received
func (a *PaxosNode) ReceiveAccept(AcceptMsg *paxos.PaxosMessage, slot *paxos.Slot) error {
	var newMsg paxos.PaxosMessage

	if AcceptMsg.ProposalNumber >= slot.Nhighest {
		slot.Nhighest = AcceptMsg.ProposalNumber
		slot.Naccepted = AcceptMsg.ProposalNumber
		slot.Vaccepted = AcceptMsg.Value
		newMsg = paxos.PaxosMessage{Type: paxos.AcceptOK, SlotNumber: AcceptMsg.SlotNumber, Value: slot.Vaccepted, ProposalNumber: AcceptMsg.ProposalNumber, ProposerID: slot.NodeInfo.NodeID}
		//write to slot here
		entry := new(paxos.LogEntry)
		entry.SlotNumber = slot.CurrentSlotNumber
		entry.ProposerID = AcceptMsg.ProposerID
		entry.Vaccepted = AcceptMsg.Value
		entry.NumberOfPlayers = len(slot.ClusterMembers)
		entry.Committed = false

		buffer, _ := json.Marshal(entry)

		a.writeAtSlot(slot.CurrentSlotNumber, buffer)
	} else {
		newMsg = paxos.PaxosMessage{Type: paxos.AcceptReject, SlotNumber: AcceptMsg.SlotNumber, Value: slot.Vaccepted, ProposalNumber: AcceptMsg.ProposalNumber, ProposerID: slot.NodeInfo.NodeID}
	}
	buf, err := json.Marshal(newMsg)
	hostPort := a.findClusterMemberHostPort(AcceptMsg.ProposerID, slot)
	if hostPort == "" {
		return fmt.Errorf("No such cluster member?")
	}
	conn, err := net.Dial("tcp", hostPort)
	if err != nil {
		return err
	}

	_, err = conn.Write(buf) // send reply to the proposer
	if err != nil {
		return err
	}
	return nil
}

///////////////////////////////////////QUIESCE///////////////////////////////////////////////////////////
// used to recreate the in-memory state from the paxos log for dead nodes that are coming back into the cluster
func (n *PaxosNode) RecreateState(cluster []utils.NodeInfo, nodeID uint32, hostport string) ([]*paxos.Slot, error) {
	fmt.Println(n)
	readLog, _ := os.OpenFile(n.logFileName, os.O_RDWR, 0666)
	var marshalledLogEntry []byte

	var i int
	for i = 0; i < MAX_SLOTS; i++ {

		readLog.Seek(int64(i*MAX_SLOT_BYTES), 0)
		var buf []byte
		buf = make([]byte, 1)
		marshalledLogEntry = make([]byte, 0)
		for {
			_, err := readLog.Read(buf[0:1])
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, err // if you return error
			} else {
				marshalledLogEntry = append(marshalledLogEntry, buf[0])
				//fmt.Printf("%c", buf[0])
				if buf[0] == '\n' {
					break
				}
			}
		}
		if len(marshalledLogEntry) == 1 {
			break
		}

		var entry *paxos.LogEntry
		err := json.Unmarshal(marshalledLogEntry, &entry)
		if err != nil {
			fmt.Printf(err.Error())
			return nil, err
		}

		n.slots[i].Committed = entry.Committed
		n.slots[i].CurrentSlotNumber = entry.SlotNumber
		n.slots[i].Vaccepted = entry.Vaccepted
	}
	n.readSlot = i

	return nil, nil
}

// quiesce operation
func (n *PaxosNode) StartQuiesce(deadNodeId uint32) {
	if n.Quiesce() {
		n.UpdateNode(deadNodeId)
		n.Restart()
	}
}

// quiescing on the healthy nodes to get everyone to the same stage and then help in adding a dead node back
func (n *PaxosNode) Quiesce() bool {
	n.QuiesceStarted = true
	fmt.Println("Quiescing..............")
	// flush pending commands
	if n.isProposing {
		<-n.QuiesceWait
	}
	fmt.Println("flushed messages")
	// fill gaps
	for i := 0; i < n.highestSlotNumber; i++ {
		slot := n.slots[i]
		if slot.Committed == false {
			fmt.Println("slot uncommitted")
			n.StartPaxosRoundForSlot(i, "") // start a nil-op
			<-n.NilOPWait
		}
	}
	fmt.Println("all nil slots covered")
	// notify if all this was successful
	return true
}

func (n *PaxosNode) UpdateNode(deadNodeId uint32) bool {
	return false
}

// marks end of quiesce
func (n *PaxosNode) Restart() bool {
	n.QuiesceStarted = false
	return true
}

///////////////////////////////////////QUIESCE///////////////////////////////////////////////////////////
