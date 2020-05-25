package paxos

import "github.com/pears-to-peers/utils"

type MessageType int

const (
	Prepare MessageType = iota + 1
	PrepareReject
	PrepareOK
	Accept
	AcceptReject
	AcceptOK
	Commit
	Quiesce
)

type PaxosMessage struct {
	Type           MessageType
	SlotNumber     int
	Value          string // proposed value
	Nhighest       uint32 // highest proposal number seen
	Naccepted      uint32 // accepted proposal number
	Vaccepted      string // accepted value
	ProposalNumber uint32 // highest proposal number seen
	ProposerID     uint32
}

type LogEntry struct {
	SlotNumber      int
	ProposerID      uint32
	Vaccepted       string
	NumberOfPlayers int
	Committed       bool
}

type Slot struct {
	// all this info is required by proposer and acceptor. where should we put it?
	ClusterMembers             []utils.NodeInfo
	NodeInfo                   *utils.NodeInfo
	CurrentSlotNumber          int    //TODO: Where is this updated
	Nhighest                   uint32 // highest proposal number seen
	Naccepted                  uint32 // accepted proposal number for current slot number
	Vaccepted                  string // accepted value for current slot number
	MyValue                    string
	NumberOfPrepareOK          int
	NumberOfAcceptOK           int
	CurrentWriteOffset         int64
	AcceptOKIDs                map[uint32]bool
	PrepareOKIDs               map[uint32]bool
	MajorityReceivedForPrepare chan bool
	MajorityReceivedForAccept  chan bool
	ProposalRejected           chan bool
	Committed                  bool
}
