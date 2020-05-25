//TODO: worry about the server cleaner

package servertests

import (
	"errors"
	"fmt"
	"github.com/pears-to-peers/centralServer"
	"github.com/pears-to-peers/rpc/centralrpc"
	"net/rpc"
	"strconv"
	"testing"
	"time"
)

const MAX_PLAYERS int = 8
const REFRESH_TIME int = 2

/*type client interface{
    conn *rpc.Client
    myID int

    startClient(ServerHostPort string) error
    joinGame(MyHostPort string, PlayerID int) (centralrpc.GetArgsReply, error)
    removeGame(MyHostPort string, PlayerID int) (centralrpc.GetArgsReply, error)
    startGame(MyHostPort string, PlayerID int) (centralrpc.GetArgsReply, error)
    stopGame(MyHostPort string, PlayerID int) (centralrpc.GetArgsReply, error)
}*/

type Client struct {
	conn *rpc.Client
	myID uint32
}

func newClient() *Client {
	newClient := new(Client)
	newClient.myID = 0
	return newClient
}

func (c *Client) startClient(ServerHostPort string) error {
	var err error
	c.conn, err = rpc.DialHTTP("tcp", ServerHostPort)

	if err != nil {
		return err
	}

	return nil
}

func (c *Client) joinGame(MyHostPort string) (centralrpc.GetArgsReply, error) {
	JoinMsg := &centralrpc.GetArgs{Hostport: MyHostPort, PlayerID: c.myID}
	var RecvMsg centralrpc.GetArgsReply

	err := c.conn.Call("centralServer.AddPlayer", JoinMsg, &RecvMsg)

	if err != nil {
		return RecvMsg, err
	}
    
    if RecvMsg.Status == centralrpc.OK{
	    c.myID = RecvMsg.PlayerID
	}
	
	return RecvMsg, nil
}

func (c *Client) removeGame(MyHostPort string) (centralrpc.GetArgsReply, error) {
	RemoveMsg := &centralrpc.GetArgs{Hostport: MyHostPort, PlayerID: c.myID}
	var RecvMsg centralrpc.GetArgsReply

	err := c.conn.Call("centralServer.RemovePlayer", RemoveMsg, &RecvMsg)

	if err != nil {
		return RecvMsg, err
	}

    if RecvMsg.Status == centralrpc.OK{
	    c.myID = 0
	}
	
	return RecvMsg, nil
}

func (c *Client) startGame(MyHostPort string) (centralrpc.GetArgsReply, error) {
	StartMsg := &centralrpc.GetArgs{Hostport: MyHostPort, PlayerID: c.myID}
	var RecvMsg centralrpc.GetArgsReply

	err := c.conn.Call("centralServer.StartPlayer", StartMsg, &RecvMsg)

	if err != nil {
		return RecvMsg, err
	}

	return RecvMsg, nil
}

func (c *Client) stopGame(MyHostPort string) (centralrpc.GetArgsReply, error) {
	StopMsg := &centralrpc.GetArgs{Hostport: MyHostPort, PlayerID: c.myID}
	var RecvMsg centralrpc.GetArgsReply

	err := c.conn.Call("centralServer.StopPlayer", StopMsg, &RecvMsg)

	if err != nil {
		return RecvMsg, err
	}

	return RecvMsg, nil
}

func (c *Client) StillAlive(MyHostPort string) (centralrpc.GetArgsReply, error) {
	StopMsg := &centralrpc.GetArgs{Hostport: MyHostPort, PlayerID: c.myID}
	var RecvMsg centralrpc.GetArgsReply

	err := c.conn.Call("centralServer.StillAlive", StopMsg, &RecvMsg)

	if err != nil {
		return RecvMsg, err
	}

	return RecvMsg, nil
}

func WaitForStart(c *Client, status chan centralrpc.Status, start chan int, correct int) {
	<-start //wait for signal to start
	reply, err := c.startGame("blah")
	if err != nil {
		return
	}

	status <- reply.Status
}

func joinTester(num_players int) error {
	var RecvMsg centralrpc.GetArgsReply

	newServer, err := centralServer.NewCentralServer()

	port := 9006
	ServerHostPort := "localhost:" + strconv.Itoa(port)

	err = newServer.Start(ServerHostPort)
	if err != nil {
		return err
	}

	//send join request
	for i := 0; i < num_players; i++ {
		c := newClient()
		err = c.startClient(ServerHostPort)

		if err != nil {
			return err
		}

		RecvMsg, err = c.joinGame("blah")

		if err != nil {
			return err
		}

		if RecvMsg.Status != centralrpc.OK {
			newServer.CloseServer()
			return errors.New("Could not add player!\n")
		}

		RecvMsg, err = c.joinGame("blah")

		if err != nil {
			return err
		}

		if RecvMsg.Status != centralrpc.UserExists {
			newServer.CloseServer()
			return errors.New("Server did not add player successfully!\n")
		}
	}

	newServer.CloseServer()
	return nil
}

func removeTester(num_players int) error {
	var err error
	var RecvMsg centralrpc.GetArgsReply

	newServer, err := centralServer.NewCentralServer()

	if err != nil {
		return err
	}

	port := 9006
	ServerHostPort := "localhost:" + strconv.Itoa(port)

	newServer.Start(ServerHostPort)

	//send join request
	for i := 0; i < num_players; i++ {
		c := newClient()
		err = c.startClient(ServerHostPort)

		if err != nil {
			return err
		}

		RecvMsg, err = c.joinGame("blah")

		if err != nil {
			return err
		}

		if RecvMsg.Status != centralrpc.OK {
			newServer.CloseServer()
			return errors.New("Could not add player!\n")
		}

		RecvMsg, err = c.removeGame("blah")

		if err != nil {
			return err
		}

		if RecvMsg.Status != centralrpc.OK {
			newServer.CloseServer()
			return errors.New("Server did not delete successfully!\n")
		}

		RecvMsg, err = c.joinGame("blah")

		if RecvMsg.Status != centralrpc.OK {
			newServer.CloseServer()
			return errors.New("Server did not delete player successfully!\n")
		}

	}

	newServer.CloseServer()
	return nil
}

func startTester(num_players int) error {
	var err error
	var RecvMsg centralrpc.GetArgsReply

	newServer, err := centralServer.NewCentralServer()

	if err != nil {
		return err
	}

	port := 9006
	ServerHostPort := "localhost:" + strconv.Itoa(port)

	newServer.Start(ServerHostPort)

	//send join request
	for i := 0; i < num_players; i++ {
		c := newClient()
		err = c.startClient(ServerHostPort)

		if err != nil {
			return err
		}

		RecvMsg, err = c.joinGame("blah")

		if err != nil {
			return err
		}

		if RecvMsg.Status != centralrpc.OK {
			newServer.CloseServer()
			return errors.New("Could not add player!\n")
		}

		RecvMsg, err = c.startGame("blah")

		if err != nil {
			return err
		}

		if RecvMsg.Status != centralrpc.OK {
			newServer.CloseServer()
			return errors.New("Request to start not successful!\n")
		}

		RecvMsg, err = c.startGame("blah")

		if RecvMsg.Status != centralrpc.OK {
			newServer.CloseServer()
			return errors.New("Request to start not successful!\n")
		}

	}

	newServer.CloseServer()
	return nil
}

/*func TestRegister (t *testing.T){
    err := centralServer.Register()
    if err != nil{
        t.Error(err)
    }
    fmt.Printf("PASS\n")
}*/

/*func TestNewServer(t *testing.T) {
	newServer, err := centralServer.NewCentralServer()

	if err != nil{
	    t.Error(err)
	}

	port := 9006
	HostPort := "localhost:" + strconv.Itoa(port)

    err = newServer.Start(HostPort)

	if err != nil{
	    t.Error(err)
	    return
	}

	err = newServer.CloseServer()

	if err != nil{
	    t.Error(err)
	    return
	}

	fmt.Printf("PASS\n")
}

func TestClose(t *testing.T) {
    newServer, err := centralServer.NewCentralServer()

	if err != nil{
	    t.Error(err)
	}

	port := 9006
	HostPort := "localhost:" + strconv.Itoa(port)

    err = newServer.Start(HostPort)

    if err != nil{
	    t.Error(err)
	    return
	}

    err = newServer.CloseServer()

    if err != nil{
	    t.Error(err)
	    return
	}

	fmt.Printf("PASS\n")
}

func TestMultipleStartClose(t *testing.T){
    newServer, err := centralServer.NewCentralServer()

	if err != nil{
	    t.Error(err)
	}

    port := 9006
	HostPort := "localhost:" + strconv.Itoa(port)

    for i:=0; i<10; i++{
        err = newServer.Start(HostPort)
        if err != nil{
            t.Error(err)
            return
        }
        err = newServer.CloseServer()
        if err != nil{
            t.Error(err)
            return
        }
    }

	fmt.Printf("PASS\n")
}

func TestJoin(t *testing.T){
    err := joinTester(1)
    if err != nil{
        t.Error(err)
        return
    }

    err = joinTester(2)
    if err != nil{
        t.Error(err)
        return
    }

    err = joinTester(5)
    if err != nil{
        t.Error(err)
        return
    }

    fmt.Printf("PASS\n")
}

func TestRemove(t *testing.T){
    err := removeTester(1)
    if err != nil{
        t.Error(err)
        return
    }

    err = removeTester(2)
    if err != nil{
        t.Error(err)
        return
    }

    err = removeTester(5)
    if err != nil{
        t.Error(err)
        return
    }

    fmt.Printf("PASS\n")
}

func TestStartGame(t *testing.T){
    newServer, err := centralServer.NewCentralServer()

	if err != nil{
	    t.Error(err)
	}

    testTime := 5

	port := 9006
	ServerHostPort := "localhost:" + strconv.Itoa(port)

    newServer.Start(ServerHostPort)

    var count int = 0

    var status centralrpc.Status
    statusUpdate := make(chan centralrpc.Status, MAX_PLAYERS)
    timeout := time.After(time.Duration(testTime) * time.Second)
    start := make(chan int) //signals when everyone should send start signal

    //start all players and make them all send ready signals
    for i:=0; i<MAX_PLAYERS; i++{
        c := newClient()
        c.startClient(ServerHostPort)
        c.joinGame("blah")
        go WaitForStart(c, statusUpdate, start)
    }

    close(start)

    for{
        select{
            case status =<-statusUpdate:
                if status == centralrpc.StartGame{
                     fmt.Printf("%d\n", count)
                    count+=1
                    if count == MAX_PLAYERS{
                        fmt.Printf("PASS\n")
                        return
                    }
                }

            case <-timeout:
                t.Error(errors.New("Test Timed Out!\n"))
                return
        }
    }
}
*/

func WaitForStop(c *Client, status chan centralrpc.Status, start chan int) {
	<-start //wait for signal to start

	var reply centralrpc.GetArgsReply
	var err error

	for {
		reply, err = c.stopGame("blah")

		if err != nil {
			return
		}

		if reply.Status == centralrpc.Stopped {
			break
		}
	}

	status <- reply.Status
}

func WaitForRemove(c *Client, status chan centralrpc.Status, start chan int) {
	<-start //wait for signal to start

	var reply centralrpc.GetArgsReply
	var err error

	for {
		reply, err = c.removeGame("blah")

		if err != nil {
			return
		}

		if reply.Status == centralrpc.OK{
			break
		}
	}

	status <- reply.Status
}
/*
func TestStartStop1(t *testing.T) {
	newServer, err := centralServer.NewCentralServer()

	if err != nil {
		t.Error(err)
	}

	delayTime := 3
	testTime := delayTime + 5

	port := 9006
	ServerHostPort := "localhost:" + strconv.Itoa(port)

	newServer.Start(ServerHostPort)

	var count int = 0

	var status centralrpc.Status
	statusUpdate := make(chan centralrpc.Status, MAX_PLAYERS)
	timeout := time.After(time.Duration(testTime) * time.Second)
	clients := make([]*Client, MAX_PLAYERS)
	start := make(chan int) //signals when everyone should send start signal

		//start half players, but don't send ready signal
	for i := 0; i < MAX_PLAYERS/2; i++ {
		c := newClient()
		clients[i] = c
		c.startClient(ServerHostPort)
		c.joinGame("blah")
	}

	//start half players and make them all send ready signals
	for i := 0; i < MAX_PLAYERS-MAX_PLAYERS/2; i++ {
		c := newClient()
		clients[i+MAX_PLAYERS/2] = c
		c.startClient(ServerHostPort)
		c.joinGame("blah")
		go WaitForStart(c, statusUpdate, start)
	}

	close(start)
	start = make(chan int)

	for i := 0; i < MAX_PLAYERS-MAX_PLAYERS/2; i++ {
		go WaitForStop(clients[i+MAX_PLAYERS/2], statusUpdate, start)
	}

	close(start)
	//make sure everyone has sent their start request
	start = make(chan int)

WAITFORSTOP:
	for {
		select {
		case status = <-statusUpdate:
			if status == centralrpc.Stopped {
				count += 1
				if count == MAX_PLAYERS-MAX_PLAYERS/2 {
					break WAITFORSTOP
				}
			}

		case <-timeout:
			t.Error(errors.New("Test Timed Out!\n"))
			return
		}
	}

	count = 0

	for i := 0; i < MAX_PLAYERS; i++ {
		go WaitForStart(clients[i], statusUpdate, start)
	}

	close(start)

	for {
		select {
		case status = <-statusUpdate:
			if status == centralrpc.StartGame {
				count += 1
				if count == MAX_PLAYERS {
					fmt.Printf("PASS\n")
					return
				}
			}

		case <-timeout:
			t.Error(errors.New("Test Timed Out!\n"))
			return
		}
	}
}
*/
func TestStartStop2(t *testing.T) {
    
}

func TestStartRemove(t *testing.T) {
    newServer, err := centralServer.NewCentralServer()

	if err != nil {
		t.Error(err)
	}

	delayTime := 3
	testTime := delayTime + 5

	port := 9006
	ServerHostPort := "localhost:" + strconv.Itoa(port)

	newServer.Start(ServerHostPort)

	var count int = 0

	var status centralrpc.Status
	statusUpdate := make(chan centralrpc.Status, MAX_PLAYERS)
	timeout := time.After(time.Duration(testTime) * time.Second)
	clients := make([]*Client, MAX_PLAYERS)
	start := make(chan int) //signals when everyone should send start signal

	//start half players, but don't send ready signal
	for i := 0; i < MAX_PLAYERS/2; i++ {
		c := newClient()
		clients[i] = c
		c.startClient(ServerHostPort)
		c.joinGame("blah")
	}

	//start half players and make them all send ready signals
	fmt.Printf("STARTING HALF PLAYERS\n")
	for i := 0; i < MAX_PLAYERS-MAX_PLAYERS/2; i++ {
		c := newClient()
		clients[i+MAX_PLAYERS/2] = c
		c.startClient(ServerHostPort)
		c.joinGame("blah")
		//fmt.Printf("MY ID IS: %d\n", clients[i+MAX_PLAYERS/2].myID)
		go WaitForStart(c, statusUpdate, start, i+MAX_PLAYERS/2+1)
	}

	close(start)
	start = make(chan int)

    
	for i := 0; i < MAX_PLAYERS-MAX_PLAYERS/2; i++ {
		go WaitForRemove(clients[i+MAX_PLAYERS/2], statusUpdate, start)
	}

	close(start)
	//make sure everyone has sent their start request
	start = make(chan int)

WAITFORREMOVE:
	for {
		select {
		case status = <-statusUpdate:
			if status == centralrpc.Stopped {
				count += 1
				if count == MAX_PLAYERS-MAX_PLAYERS/2 {
					break WAITFORREMOVE
				}
			}

		case <-timeout:
			t.Error(errors.New("Test Timed Out!\n"))
			return
		}
	}
	
    //Allow removed players to rejoin and start unstarted players
    count = 0
    for i := 0; i < MAX_PLAYERS/2; i++ {
		go WaitForStart(clients[i], statusUpdate, start, i+1)
	}
	
    for i := 0; i < MAX_PLAYERS-MAX_PLAYERS/2; i++ {
		clients[i+MAX_PLAYERS/2].joinGame("blah")
		//fmt.Printf("MY ID IS: %d\n", clients[i+MAX_PLAYERS/2].myID)
		go WaitForStart(clients[i+MAX_PLAYERS/2], statusUpdate, start, 8+i+1)
	}
    
	close(start)
    
	for{
		select {
		case status = <-statusUpdate:
		    if status == centralrpc.Stopped{
		    fmt.Printf("STATUS: %d\n", status)
		    }
			if status == centralrpc.StartGame {    
				count += 1
				if count == MAX_PLAYERS {
					fmt.Printf("PASS\n")
					return
				}
			}

		case <-timeout:
			t.Error(errors.New("Test Timed Out!\n"))
			return
		}
	}
}
