package acceptor

import (
	"github.com/pears-to-peers/paxos"
)

type acceptor struct {
	commonInfo *paxos.Slot
}

func NewAcceptor(commonInfo *paxos.Slot) Acceptor {
	newacceptor := new(acceptor)
	newacceptor.commonInfo = commonInfo
	return newacceptor
}

func (a *acceptor) ReceiveCommit() error {
	return nil
}
