package proposer

import ()

type Proposer interface {
	SendPrepare(slotNumber int, proposalNumber uint32) error
	// ReceivePrepareOK(OKMsg *paxos.PaxosMessage) bool
	SendAccept(value string, slotNumber int, proposalNumber uint32) error
	// ReceiveAcceptOK(OKMsg *paxos.PaxosMessage) bool
	SendCommit(value string, slotNumber int, proposalNumber uint32) error
}
