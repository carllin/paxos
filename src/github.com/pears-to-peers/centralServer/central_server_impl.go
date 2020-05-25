// Central server that manages the joining of players
package centralServer

import (
	"encoding/json"
	"fmt"
	"github.com/pears-to-peers/paxos"
	"github.com/pears-to-peers/rpc/centralrpc"
	"github.com/pears-to-peers/utils"
	"net"
	"net/http"
	"net/rpc"
	"sync"
	"time"
)

const MAX_PLAYERS int = 8
const MIN_PLAYERS int = 3

type central_server struct {
	ListenSock  net.Listener
	ProtectJoin sync.Mutex
	Players     map[uint32]centralrpc.Player
	GameStarted bool

	next_id             uint32
	num_players         int
	num_started_players int

	die chan int
}

func NewCentralServer() (centralServer, error) {
	fmt.Println("I am the Central Server")

	newCentralServer := new(central_server)
	newCentralServer.Players = make(map[uint32]centralrpc.Player)
	newCentralServer.GameStarted = false
	newCentralServer.next_id = 1
	newCentralServer.num_players = 0
	newCentralServer.num_started_players = 0
	return newCentralServer, nil
}

func (c *central_server) SendQuiesceMessage() {
	var nodeId uint32
	fmt.Println("Which node do you want to bring back?")
	fmt.Scanf("%d\n", &nodeId)
	msg := new(paxos.PaxosMessage)
	msg.Type = paxos.Quiesce
	msg.ProposerID = nodeId
	buf, _ := json.Marshal(msg)
	for _, v := range c.Players {
		conn, err := net.Dial("tcp", v.Hostport)
		if err == nil {
			fmt.Printf("sending message to node %d : %s", v.Hostport, string(buf))
			_, _ = conn.Write(buf)
		}
	}
}

func checkAlive(c *central_server) {
	interval := 2 //interval in seconds
	alarmChan := time.After(time.Duration(interval) * time.Second)

	for {
		select {
		case <-c.die:
			return
		case <-alarmChan:
			checkAliveHelper(c, interval)
			alarmChan = time.After(time.Duration(interval) * time.Second)
		}
	}
}

func checkAliveHelper(c *central_server, timeout int) {
	c.ProtectJoin.Lock()
	for k, v := range c.Players {
		//check if player is still alive
		if v.LastMsg.Add(time.Duration(timeout) * time.Second).Before(time.Now()) {
			DeletePlayer(c, k)
		}
	}
	c.ProtectJoin.Unlock()
}

func (c *central_server) Start(port string) error {
	c.next_id = 1
	c.die = make(chan int)

	//register RPC
	err := rpc.RegisterName("centralServer", centralrpc.Wrap(c))

	if err != nil {
		return err
	}

	rpc.HandleHTTP()

	c.ListenSock, err = net.Listen("tcp", port)

	if err != nil {
		return err
	}

	go http.Serve(c.ListenSock, nil) // we are now ready to receive RPC requests

	//start goroutine to check players to see if they're still in room

	//go checkAlive(c) TODO: Replace this after initial tests
	return nil
}

func (c *central_server) CloseServer() error {
	c.ProtectJoin.Lock()

	err := (c.ListenSock).Close()

	//rpc.UnregisterName("centralServer")

	if err != nil {
		return err
	}

	close(c.die)

	for player, _ := range c.Players {
		DeletePlayer(c, player)
	}

	c.GameStarted = false
	c.ProtectJoin.Unlock()

	return nil
}

func (c *central_server) AddPlayer(args *centralrpc.GetArgs, reply *centralrpc.GetArgsReply) error {
	c.ProtectJoin.Lock()
	if c.num_players == MAX_PLAYERS {
		reply.Status = centralrpc.RoomFull
		c.ProtectJoin.Unlock()
		return nil
	}

	if c.GameStarted == true {
		reply.Status = centralrpc.GameStarted
		c.ProtectJoin.Unlock()
		return nil
	}

	if args.PlayerID != 0 { //worry about security later, go badly with corrupted msgs
		player := c.Players[args.PlayerID]
		player.LastMsg = time.Now()
		c.Players[args.PlayerID] = player
		reply.Status = centralrpc.UserExists
		c.ProtectJoin.Unlock()
		return nil
	}

	c.num_players += 1
	reply.Status = centralrpc.OK
	reply.PlayerID = c.next_id
	c.Players[c.next_id] = NewPlayer(args.Hostport, c.next_id) //protected by mutex, so no overlapping numbers
	c.next_id += 1
	c.ProtectJoin.Unlock()
	fmt.Println("Welcome to the game player", reply.PlayerID)
	return nil
}

func (c *central_server) RemovePlayer(args *centralrpc.GetArgs, reply *centralrpc.GetArgsReply) error {
	c.ProtectJoin.Lock()
	//mite have to send kill signal on blocking "waitforgame" rpc call if we're doing it that way
	if c.GameStarted == true {
		reply.Status = centralrpc.GameStarted
		c.ProtectJoin.Unlock()
		return nil
	}

	if _, ok := c.Players[args.PlayerID]; !ok {
		reply.Status = centralrpc.NoSuchUser
		c.ProtectJoin.Unlock()
		return nil
	}

	DeletePlayer(c, args.PlayerID)
	reply.Status = centralrpc.OK
	c.ProtectJoin.Unlock()
	return nil
}

func DeletePlayer(c *central_server, PlayerID uint32) { //make sure to lock before calling this
	c.num_players -= 1 //TODO: Decide if a player leaves if we should reset everyone's decision to "start". Right now, answer is no
	if c.Players[PlayerID].Started == true {
		c.num_started_players -= 1
		close(c.Players[PlayerID].StopWait)
	}

	delete(c.Players, PlayerID)
	return
}

func (c *central_server) StopPlayer(args *centralrpc.GetArgs, reply *centralrpc.GetArgsReply) error {
	c.ProtectJoin.Lock()
	if c.GameStarted == true {
		reply.Status = centralrpc.GameStarted
		c.ProtectJoin.Unlock()
		return nil
	}

	if c.Players[args.PlayerID].Started != true {
		reply.Status = centralrpc.OK
		c.ProtectJoin.Unlock()
		return nil
	}

	player := c.Players[args.PlayerID]
	c.Players[args.PlayerID].StopWait <- 1
	c.num_started_players -= 1
	player.Started = false
	c.Players[args.PlayerID] = player
	reply.Status = centralrpc.OK
	c.ProtectJoin.Unlock() //can't unlock in case of removeplayer
	return nil

}

func (c *central_server) StartPlayer(args *centralrpc.GetArgs, reply *centralrpc.GetArgsReply) error {
	c.ProtectJoin.Lock()

	val, ok := c.Players[args.PlayerID]

	if !ok {
		reply.Status = centralrpc.NoSuchUser
		c.ProtectJoin.Unlock() //no such user
		return nil
	}

	player := c.Players[args.PlayerID]
	player.LastMsg = time.Now()
	c.Players[args.PlayerID] = player

	if val.Started == true {
		reply.Status = centralrpc.OK
		c.ProtectJoin.Unlock()
		return nil
	}

	player.Started = true
	c.Players[args.PlayerID] = player
	c.num_started_players += 1

	if c.num_started_players == c.num_players && c.num_players >= MIN_PLAYERS {
		c.GameStarted = true //no other rpc calls can run
		close(c.die)
		StartGame(c.Players)
	}

	c.ProtectJoin.Unlock()

	fmt.Println("waiting for minimum number of players to join")
	select {
	case <-c.Players[args.PlayerID].WaitForStart:
		reply.Status = centralrpc.StartGame
		nodecluster := make([]utils.NodeInfo, 0, len(c.Players))

		for _, v := range c.Players {
			nodecluster = append(nodecluster, centralrpc.PlayertoNode(v))
		}
		fmt.Println("We have enough players to start :D")
		reply.AllPlayers = nodecluster
	case <-c.Players[args.PlayerID].StopWait:
		reply.Status = centralrpc.Stopped
	}

	return nil
}

func (c *central_server) StillAlive(args *centralrpc.GetArgs, reply *centralrpc.GetArgsReply) error {
	player := c.Players[args.PlayerID]
	player.LastMsg = time.Now()
	c.Players[args.PlayerID] = player
	reply.Status = centralrpc.OK
	return nil
}

func StartGame(PlayerList map[uint32]centralrpc.Player) error { //TODO
	for _, player := range PlayerList {
		player.WaitForStart <- 1
	}

	return nil
}

func NewPlayer(HostPort string, playerID uint32) centralrpc.Player {
	newPlayer := new(centralrpc.Player)
	newPlayer.Hostport = HostPort
	newPlayer.PlayerID = playerID
	newPlayer.WaitForStart = make(chan int, 1)
	newPlayer.StopWait = make(chan int)
	newPlayer.Started = false
	newPlayer.LastMsg = time.Now()
	return *newPlayer
}
