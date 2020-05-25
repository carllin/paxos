package centralrpc

import (
	"github.com/pears-to-peers/utils"
	"time"
)

type Status int

const (
	OK     Status = iota + 1 //creates a separate type from int, The RPC was a success.
	NoGame                   // The game does not exist
	UserExists
	NoSuchUser
	RoomFull
	StartGame
	Stopped
	GameStarted //cannot join game because game has already started
)

type RemoteCentralServer interface {
	//GetPlayers(*GetArgs, *GetArgsReply) error
	AddPlayer(args *GetArgs, reply *GetArgsReply) error
	RemovePlayer(args *GetArgs, reply *GetArgsReply) error
	StartPlayer(args *GetArgs, reply *GetArgsReply) error
	StopPlayer(args *GetArgs, reply *GetArgsReply) error
	StillAlive(args *GetArgs, reply *GetArgsReply) error
}

type CentralServer struct {
	RemoteCentralServer
}

func Wrap(c RemoteCentralServer) RemoteCentralServer {
	return &CentralServer{c}
}

type Player struct {
	Hostport     string
	PlayerID     uint32
	Started      bool
	WaitForStart chan int
	StopWait     chan int
	LastMsg      time.Time
}

type Node struct {
	Hostport string
	PlayerID uint32
}

type GetArgs struct {
	Hostport string
	PlayerID uint32 //remember to send this! when you don't have one, use zero
}

type GetArgsReply struct {
	Status     Status
	PlayerID   uint32
	AllPlayers []utils.NodeInfo
}

func PlayertoNode(Player Player) utils.NodeInfo {
	newNode := new(utils.NodeInfo)
	newNode.HostPort = Player.Hostport
	newNode.NodeID = Player.PlayerID
	return *newNode
}
