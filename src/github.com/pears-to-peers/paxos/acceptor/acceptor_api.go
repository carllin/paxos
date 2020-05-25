package acceptor

import ()

type Acceptor interface {
	// ReceivePrepare(PrepMsg *paxos.PaxosMessage, conn net.Conn) error
	// ReceiveAccept(PrepMsg *paxos.PaxosMessage, conn net.Conn) error
	ReceiveCommit() error
	//SendAcceptOK(value string, slotNumber int, proposalNumber uint32) error
	//SendPrepareOK(value string, slotNumber int, proposalNumber uint32) error
}
