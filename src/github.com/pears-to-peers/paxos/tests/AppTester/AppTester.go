package main

import (
	//"fmt"
	//"github.com/pears-to-peers/utils"
	//"github.com/pears-to-peers/paxos/paxosnode"
	//"github.com/pears-to-peers/paxos/proposer"
	"github.com/pears-to-peers/cli"
	"github.com/pears-to-peers/centralServer"
	"strconv"
	"math/rand"
	"os"
	//"errors"
	//"io"
	//"time"
	//"bufio"
    "fmt"
)

const(
    MAX_PLAYERS int = 8
)

func TStartGame(cli *cli.Cli, finish chan bool){
    cli.StartGame()
    finish <- true
}

func TStartLeaderElection(cli *cli.Cli, finish chan bool){
    cli.StartLeaderElection()
    finish <- true
}

func TChooseTopic(cli *cli.Cli, finish chan bool){
    cli.ChooseTopic()
    finish <- true
}

func TSubmitContent(cli *cli.Cli, finish chan bool){
    cli.SubmitContent()
    finish <- true
}

func TPickWinner(cli *cli.Cli, finish chan bool){
    cli.PickWinner()
    finish <- true
}

/*func CheckLogConsistent(cli []*cli.Cli, numPlayers int) error{ //assume no network failure
    files := make([]*bufio.Reader, MAX_PLAYERS)
    
    for i:=0; i<numPlayers; i++{
        fmt.Printf("%d\n", i)
        filename := "/tmp/paxosLog_"+strconv.Itoa(int(cli[i].PlayerID))
        file, err := os.OpenFile(filename, os.O_RDWR, 0666)
        if err != nil{
            return err
        }
        files[i] = bufio.NewReader(file)
    } 
    
    for{
        mainstring, err := files[0].ReadString('\n')
        if err == io.EOF{
            for i:=1; i<numPlayers; i++{
                _, err := files[i].ReadString('\n')
                if err != io.EOF{
                    return errors.New("LOGS INCONSISTENT: SOMEONE HAS SHORTER LOG\n")
                }
            }
            return nil
        }
        
        for i:=1; i<numPlayers; i++{
            teststring, err := files[i].ReadString('\n')
            if err != nil{
                return err
            }
            if teststring != mainstring{
                return errors.New("LOGS INCONSISTENT\n")
            }
        }
    }
    
    return nil
}*/

func main(){
    switch os.Args[1]{
        case "1":
           TestBasicLeader()
        case "2":
           TestLeaderFail()
        case "3":
            TestLeaderFail2()
    }
}

func TestBasicLeader() error{
    Finish := make(chan bool)
    num_players := 3
    count := 0
    clients := make([]*cli.Cli, MAX_PLAYERS)
    newServer, err := centralServer.NewCentralServer()
    
	if err != nil {
		return err
	}

	port := 9006
	ServerHostPort := "localhost:" + strconv.Itoa(port)

	newServer.Start(ServerHostPort)

	for i:=0; i<num_players; i++{
	    clientport := strconv.Itoa(int(rand.Int31n(10000)+10000))
	    clients[i], _ = cli.NewCli(ServerHostPort, ":" + clientport)
	    clients[i].JoinGame()
	}   
	
	for i:=0; i<num_players; i++{
	    go TStartGame(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    go TStartLeaderElection(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
    
    fmt.Printf("PASS\n")	
	
	return nil
}

//leader fails during topic selection
func TestLeaderFail() error{
    Finish := make(chan bool)
    num_players := 3
    count := 0
    clients := make([]*cli.Cli, MAX_PLAYERS)
    newServer, err := centralServer.NewCentralServer()
    
	if err != nil {
		return err
	}

	port := 9006
	ServerHostPort := "localhost:" + strconv.Itoa(port)

	newServer.Start(ServerHostPort)

	for i:=0; i<num_players; i++{
	    clientport := strconv.Itoa(int(rand.Int31n(10000)+10000))
	    clients[i], _ = cli.NewCli(ServerHostPort, ":" + clientport)
	    clients[i].JoinGame()
	}
	
	for i:=0; i<num_players; i++{
	    go TStartGame(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    go TStartLeaderElection(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
    
	for i:=0; i<num_players; i++{
	    //leader terminates
	    if clients[i].IsJudge == true{
	        continue
	    }
	    
	    go TChooseTopic(clients[i], Finish) //should select new leader who picks new topic
	}
	
	for{
	    <-Finish
        count+=1
	    if count == num_players-1{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    //leader terminates
	    if clients[i].IsJudge == true{
	        continue
	    }
	    
	    go TSubmitContent(clients[i], Finish)
	}
	
	for{
	    <-Finish
        count+=1
	    if count == num_players-1{
	        count = 0
	        break
	    }
	}
	
	//TODO: Check log here
    
    fmt.Printf("PASS\n")
	return nil
}

//leader fails while receiving submissions
func TestLeaderFail2() error{
    Finish := make(chan bool)
    num_players := 3
    count := 0
    clients := make([]*cli.Cli, MAX_PLAYERS)
    newServer, err := centralServer.NewCentralServer()
    
	if err != nil {
		return err
	}

	port := 9006
	ServerHostPort := "localhost:" + strconv.Itoa(port)

	newServer.Start(ServerHostPort)

	for i:=0; i<num_players; i++{
	    clients[i], _ = cli.NewCli(ServerHostPort, string(rand.Int31n(10000)+10000))
	    clients[i].JoinGame()
	}
	
    for i:=0; i<num_players; i++{
	    clientport := strconv.Itoa(int(rand.Int31n(10000)+10000))
	    clients[i], _ = cli.NewCli(ServerHostPort, ":" + clientport)
	    clients[i].JoinGame()
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    go TStartLeaderElection(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
		
	for i:=0; i<num_players; i++{
	    go TChooseTopic(clients[i], Finish)
	}
	
	for{
	    <-Finish
        count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    if clients[i].IsJudge == true{
	        continue
	    }	  
	   
	    go TSubmitContent(clients[i], Finish)
	}
	
	for{
	    <-Finish
        count+=1
	    if count == num_players-1{
	        count = 0
	        break
	    }
	}
	
    fmt.Printf("PASS\n")
    
	return nil
}


func NormalFullRound() error{
    Finish := make(chan bool)
    num_players := 3
    count := 0
    clients := make([]*cli.Cli, MAX_PLAYERS)
    newServer, err := centralServer.NewCentralServer()
    
	if err != nil {
		return err
	}

	port := 9006
	ServerHostPort := "localhost:" + strconv.Itoa(port)

	newServer.Start(ServerHostPort)

	for i:=0; i<num_players; i++{
	    clientport := strconv.Itoa(int(rand.Int31n(10000)+10000))
	    clients[i], _ = cli.NewCli(ServerHostPort, ":" + clientport)
	    clients[i].JoinGame()
	}
	
	for i:=0; i<num_players; i++{
	    go TStartGame(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    go TStartLeaderElection(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    go TChooseTopic(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    go TChooseTopic(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	for i:=0; i<num_players; i++{
	    go TSubmitContent(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    go TPickWinner(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
    fmt.Printf("PASS\n")
    
	return nil
}


func WinnerDies(WinnerID uint32) error{
    Finish := make(chan bool)
    num_players := 3
    count := 0
    clients := make([]*cli.Cli, MAX_PLAYERS)
    newServer, err := centralServer.NewCentralServer()
    
	if err != nil {
		return err
	}

	port := 9006
	ServerHostPort := "localhost:" + strconv.Itoa(port)

	newServer.Start(ServerHostPort)

	for i:=0; i<num_players; i++{
	    clientport := strconv.Itoa(int(rand.Int31n(10000)+10000))
	    clients[i], _ = cli.NewCli(ServerHostPort, ":" + clientport)
	    clients[i].JoinGame()
	}
	
	for i:=0; i<num_players; i++{
	    go TStartGame(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    go TStartLeaderElection(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    go TChooseTopic(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    go TChooseTopic(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	for i:=0; i<num_players; i++{
	    go TSubmitContent(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    go TPickWinner(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    if clients[i].PlayerID == WinnerID{
	        clients[i].Close()
	    }
	    
	    go TChooseTopic(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players-1{
	        count = 0
	        break
	    }
	}
    
    fmt.Printf("PASS\n")
    
	return nil
}

//Half-1 clients (still majority alive) die before submitting content (remember to make leader in second half of players!)
func HalfFailClients() error{
    Finish := make(chan bool)
    num_players := 3
    num_dead := (num_players-1)/2
    count := 0
    clients := make([]*cli.Cli, MAX_PLAYERS)
    newServer, err := centralServer.NewCentralServer()
    
	if err != nil {
		return err
	}

	port := 9006
	ServerHostPort := "localhost:" + strconv.Itoa(port)

	newServer.Start(ServerHostPort)

	for i:=0; i<num_players; i++{
	    clientport := strconv.Itoa(int(rand.Int31n(10000)+10000))
	    clients[i], _ = cli.NewCli(ServerHostPort, ":" + clientport)
	    clients[i].JoinGame()
	}
	
	for i:=0; i<num_players; i++{
	    go TStartGame(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    go TStartLeaderElection(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    go TChooseTopic(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	//close half the clients
	for i:=0; i<num_dead; i++{
	    clients[i].Close()
	}
	
	for i:=num_dead; i<num_players; i++{
	    go TSubmitContent(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players-num_dead{
	        count = 0
	        break
	    }
	}
	
	for i:=num_dead; i<num_players; i++{
	    go TPickWinner(clients[i], Finish)
	}
	
	for{
        <-Finish
        count+=1
	    if count == num_players-num_dead{
	        count = 0
	        break
	    }
	}
	
	for i:=num_dead; i<num_players; i++{
	    go TChooseTopic(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players-num_dead{
	        count = 0
	        break
	    }
	}
    
    fmt.Printf("PASS\n")
    
	return nil
}

//The leader and half-2 of the clients die after submitting content, majority remain alive
func LeaderandClientDie(WinnerID uint32) error{
    Finish := make(chan bool)
    num_players := 3
    num_dead := (num_players-1)/2
    count := 0
    clients := make([]*cli.Cli, MAX_PLAYERS)
    newServer, err := centralServer.NewCentralServer()
    
	if err != nil {
		return err
	}

	port := 9006
	ServerHostPort := "localhost:" + strconv.Itoa(port)

	newServer.Start(ServerHostPort)

	for i:=0; i<num_players; i++{
	    clientport := strconv.Itoa(int(rand.Int31n(10000)+10000))
	    clients[i], _ = cli.NewCli(ServerHostPort, ":" + clientport)
	    clients[i].JoinGame()
	}
	
	for i:=0; i<num_players; i++{
	    go TStartGame(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    go TStartLeaderElection(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    go TChooseTopic(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players; i++{
	    go TSubmitContent(clients[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players{
	        count = 0
	        break
	    }
	}
	
	//kill half-2 clients and the leader
	killed := 0
	alive := make([]*cli.Cli, num_players-num_dead)
	numalive := 0
	
	for i:=0; i<num_players; i++{
	    if clients[i].IsJudge == true{
	        clients[i].Close()
	        continue
	    }    
	    if killed != num_dead-1{
	        clients[i].Close()
	        killed +=1
	        continue
	    }
	    
	    alive[numalive] = clients[i]
	    numalive +=1
	}
	
	for i:=0; i<num_players-num_dead; i++{
	    go TPickWinner(alive[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players-num_dead{
	        count = 0
	        break
	    }
	}
	
	for i:=0; i<num_players-num_dead; i++{
	    go TChooseTopic(alive[i], Finish)
	}
	
	for{
	    <-Finish
	    count+=1
	    if count == num_players-num_dead{
	        count = 0
	        break
	    }
	}
    
    fmt.Printf("PASS\n")
    
	return nil
}
