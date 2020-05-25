package centralServer

import (
	"github.com/pears-to-peers/rpc/centralrpc"
)

type centralServer interface {
	Start(port string) error
	CloseServer() error
	AddPlayer(args *centralrpc.GetArgs, reply *centralrpc.GetArgsReply) error
	RemovePlayer(args *centralrpc.GetArgs, reply *centralrpc.GetArgsReply) error
	StartPlayer(args *centralrpc.GetArgs, reply *centralrpc.GetArgsReply) error
	StopPlayer(args *centralrpc.GetArgs, reply *centralrpc.GetArgsReply) error
	SendQuiesceMessage()
}
