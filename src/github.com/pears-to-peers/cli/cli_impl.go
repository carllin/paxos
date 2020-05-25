// Player and game logic implementer. It has a paxosNode encapsulated within it. The paxosNode is used to append to
// the game log. The paxosNode.NextMessage() is used to get back committed messages from the log in order
// and decide what to do next in the game.
package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/pears-to-peers/paxos"
	"github.com/pears-to-peers/paxos/paxosnode"
	"github.com/pears-to-peers/rpc/centralrpc"
	"github.com/pears-to-peers/utils"
	"io"
	"net/rpc"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	InputTimeout      = 10
	SubmissionTimeout = 15
	LeaderTimeout     = 60
	MAX_PLAYERS       = 8
)

type cli struct {
	serverHostPort string
	myHostPort     string
	playerID       uint32
	submitLock     sync.Mutex
	conn           *rpc.Client
	paxosNode      *paxosnode.PaxosNode
	submissions    map[uint32]string
	isJudge        bool
	isWinner       bool
	judgeID        uint32
	topic          string
	checkerSpawned bool
	progressing    bool
	killLoop       chan struct{}
	messageGetter  bool
	LogEntry       chan *paxos.LogEntry
	confFileName   string
}

func NewCli(serverHostPort string, myHostPort string) (*cli, error) {
	fmt.Println("*******************Welcome to Pears to Peers************************")
	newcli := new(cli)
	newcli.serverHostPort = serverHostPort
	newcli.myHostPort = myHostPort
	newcli.submissions = make(map[uint32]string)
	newcli.checkerSpawned = false
	newcli.progressing = false
	newcli.killLoop = make(chan struct{})
	newcli.LogEntry = make(chan *paxos.LogEntry)
	newcli.messageGetter = false
	return newcli, nil
}

func (c *cli) JoinGame() error {
	conn, errRpc := rpc.DialHTTP("tcp", c.serverHostPort)
	if errRpc != nil {
		fmt.Println("Could not dial into the centralServer")
		return errRpc
	}

	c.conn = conn
	args := centralrpc.GetArgs{}
	var reply centralrpc.GetArgsReply
	args.Hostport = c.myHostPort
	args.PlayerID = 0 // new player

	if err := c.conn.Call("centralServer.AddPlayer", &args, &reply); err != nil {
		fmt.Println("error in AddPlayer", err)
		return err
	}

	if reply.Status == centralrpc.GameStarted {
		return fmt.Errorf("Game started!! Go away!!")
	}

	fmt.Printf("You have been added to the game, Player %d!\n", reply.PlayerID)
	c.playerID = reply.PlayerID
	return nil
}

func (c *cli) StartGame() error {
	StartMsg := &centralrpc.GetArgs{Hostport: c.myHostPort, PlayerID: c.playerID}
	var RecvMsg centralrpc.GetArgsReply
	var wantToStart string

	fmt.Println("Enter s to start")
	for {
		fmt.Scanf("%s\n", &wantToStart)
		if wantToStart == "s" {
			break
		}
	}

	err := c.conn.Call("centralServer.StartPlayer", StartMsg, &RecvMsg)
	if err != nil {
		fmt.Printf("%d\n", RecvMsg.Status)
		fmt.Println(err)
		return err
	}

	switch msgType := RecvMsg.Status; {
	case msgType == centralrpc.RoomFull:
		return fmt.Errorf("Room full! come back later")
	case msgType == centralrpc.StartGame:
		playerIDs := make([]uint32, 0, len(RecvMsg.AllPlayers))
		fmt.Println("Alright! Let's play!!!...")
		for i := 0; i < len(RecvMsg.AllPlayers); i++ {
			playerIDs = append(playerIDs, RecvMsg.AllPlayers[i].NodeID)
		}
		fmt.Println("You are playing with players", playerIDs)
	case msgType == centralrpc.NoSuchUser:
		return fmt.Errorf("No such user")
	case msgType == centralrpc.OK:
		return fmt.Errorf("Game already on!!!") // TODO: allow him to get in. handle the other bits(paxos node etc)
	}
	c.paxosNode, err = paxosnode.NewPaxosNode(c.myHostPort, c.playerID, RecvMsg.AllPlayers, 0, false) // what happens if some players do this before others and start communicating?
	// fmt.Println("Paxos node created")
	c.confFileName = "/tmp/confFile_" + strconv.Itoa(int(c.playerID))

	var f *os.File
	f, _ = os.Create(c.confFileName)
	f.Close()

	return err
}

///////////////////////QUIESCE//////////////////////////////////
func (c *cli) StartQuiesce(filename string, nodeID uint32) error {
	c.playerID = nodeID
	cluster, err := c.RecreatePeers(filename)
	newNode, err := paxosnode.NewPaxosNode(c.myHostPort, c.playerID, cluster, 0, true)
	c.SetPaxosNode(newNode)
	_, err = c.paxosNode.RecreateState(cluster, c.playerID, c.myHostPort)
	return err
}

// recreates cluster for use in quiescing for the dead node
func (c *cli) RecreatePeers(filename string) ([]utils.NodeInfo, error) {
	newCluster := make([]utils.NodeInfo, 0, MAX_PLAYERS)
	file, _ := os.OpenFile(filename, os.O_RDWR, 0666)
	Reader := bufio.NewReader(file)

	for i := 0; i < MAX_PLAYERS; i++ {
		nodeinfo, err := Reader.ReadBytes('\n')
		if err == io.EOF {
			return newCluster, nil
		}

		if err != nil {
			return newCluster, nil
		}

		var entry utils.NodeInfo

		err = json.Unmarshal(nodeinfo, &entry)
		if err != nil {
			return newCluster, nil
		}
		newCluster = append(newCluster, entry)
	}
	file.Close()

	return newCluster, nil
}

///////////////////////QUIESCE//////////////////////////////////

func GetInputFromUser(userInput chan string, kill chan bool) {
	var wantsToBeJudge string
	select {
	default:
		fmt.Scanf("%s\n", &wantsToBeJudge)
		userInput <- wantsToBeJudge
	case <-kill:
		return
	}

}

func (c *cli) StartLeaderElection() error {
	fmt.Println("Do you wanna be judge for Round 1? Y/N")
	var wantsToBeJudge string

	userInput := make(chan string, 1)
	kill := make(chan bool, 1)
	go GetInputFromUser(userInput, kill)
	select {
	case <-time.After(time.Duration(InputTimeout) * time.Second):
		fmt.Println("No Response\n")
		wantsToBeJudge = "N"
		kill <- true
	case wantsToBeJudge = <-userInput:
	}

	var contestForElection bool
	switch judge := wantsToBeJudge; {
	case judge == "Y":
		fmt.Println("Ok, so you want to be judge! You are in the election")
		contestForElection = true
	case judge == "N":
		fmt.Println("Ok, so you don't want to be judge! Hang on till we select a judge from the other players")
		contestForElection = false
	}

	if contestForElection {
		judgeStmt := fmt.Sprintf("Player %d wants to be judge", c.playerID)
		_, err := c.paxosNode.StartPaxosRoundFor(judgeStmt) //TODO: judge can be nil (rejected), then how do we know who judge is (call nextmessage instead?)
		if err != nil {
			fmt.Println("Something went wrong!!!", err)
			return err
		}
	}
	return nil
}
func (c *cli) ReadMsg(message chan string) {
	var topicEntry *paxos.LogEntry
	var topic string
	// var err error

	// fmt.Println("reading")
	for {
		topicEntry, _ = c.paxosNode.NextMessage()

		if topicEntry == nil {
			time.Sleep(1 * time.Second)
			continue
		} else {
			if num, _ := fmt.Sscanf(topicEntry.Vaccepted, "Topic chosen is %s", &topic); num != 0 { // if it is the winner message
				// found = true
				message <- topic
				break
			}
		}
	}

}

func (c *cli) AskForSubmission(Submissions chan string, kill chan bool) error {
	fmt.Println("Select a url for submission for the topic ", c.topic)
	var url string
	select {
	default:
		n, err := fmt.Scanf("%s\n", &url)
		if err != nil || n != 1 {
			fmt.Println(n, err)
			return err
		}

		Submissions <- url
		return nil
	case <-kill:
		return nil
	}
}

func (c *cli) getMessage(msg chan *paxos.LogEntry) {
	var submission *paxos.LogEntry
	var err error
	for {
		submission, err = c.paxosNode.NextMessage()
		if submission == nil {
			// fmt.Println("contuning trying to read")
			continue
		} else {
			break
		}
	}

	if err != nil {
		fmt.Println("error in next getMessage", err)
		msg <- nil
		return
	}
	msg <- submission
}

func (c *cli) ReceiveSubmissions() {
	var submission *paxos.LogEntry
	var url string
	msgchan := make(chan *paxos.LogEntry, 1)
	c.getMessage(msgchan)
	for {
		select {
		case <-time.After(time.Duration(SubmissionTimeout * time.Second)):
			return
		case submission = <-msgchan:

			if num, _ := fmt.Sscanf(submission.Vaccepted, "Submission is: %s", &url); num != 0 { //change to flag later?
				//TODO: what if receive winner msg was received here, it would get ignored
				// c.submitLock.Lock()
				fmt.Println("received submission:", url)
				c.submissions[submission.ProposerID] = url
				// c.submitLock.Unlock()
				c.getMessage(msgchan)
			}
		}
	}
}

func (c *cli) SetPaxosNode(paxosNode *paxosnode.PaxosNode) {
	c.paxosNode = paxosNode
}

func (c *cli) StartGameLoop() {
	var err error
	if c.isJudge == false {
		err = c.StartLeaderElection()
	}

	c.progressing = true
	if err != nil {
		fmt.Println("error", err)
		return
	}

	c.Loop()

}

func (c *cli) getNextMessage() {
	for {
		// fmt.Println("reading next message")
		msg, _ := c.paxosNode.NextMessage()
		// fmt.Println(msg)
		if msg == nil {
			continue
		}
		c.LogEntry <- msg
	}

}

// MAIN GAME LOOP
func (c *cli) Loop() {
	if !c.messageGetter {
		go c.getNextMessage()
		c.messageGetter = true
	}

	var winnerID uint32
	var url string
	var topic string
	var judgeID uint32
	timedOut := false
	for {
		if timedOut {
			break
		}
		c.progressing = false
		select {
		case msg := <-c.LogEntry:
			fmt.Println("----------------------------------------")
			winnerSlot, _ := fmt.Sscanf(msg.Vaccepted, "The winner is Player %d", &winnerID)
			submissionSlot, _ := fmt.Sscanf(msg.Vaccepted, "Submission is: %s", &url)
			topicSlot, _ := fmt.Sscanf(msg.Vaccepted, "Topic chosen is %s", &topic)
			judgeSlot, _ := fmt.Sscanf(msg.Vaccepted, "Player %d wants to be judge", &judgeID)
			if judgeSlot != 0 {
				c.progressing = true
				if c.judgeID != 0 && c.progressing == true {

				} else {
					if judgeID == c.playerID {
						fmt.Println("We were successfully chosen as judge")
						c.isJudge = true
						c.judgeID = c.playerID
						c.submissions = make(map[uint32]string)
					} else {
						fmt.Println("Judge chosen is ", judgeID)
						c.isJudge = false
						c.judgeID = judgeID
					}
					if c.isJudge {
						// choose the topic
						fmt.Println("Select a topic")
						var topic string
						n, err := fmt.Scanf("%s\n", &topic)
						if err != nil || n != 1 {
							fmt.Println(n, err)
						}
						topicStmt := fmt.Sprintf("Topic chosen is %s", topic)
						_, err = c.paxosNode.StartPaxosRoundFor(topicStmt) //won't have competing proposals
						if err != nil {
							fmt.Println("Something went wrong!!!", err)
						}
					}
					c.progressing = true
				}
			}
			if topicSlot != 0 {
				c.progressing = true
				c.topic = topic
				if c.isJudge {
					// receive submissions
					fmt.Printf("Waiting for submissions, players have %d seconds\n", SubmissionTimeout)
					go c.ReceiveSubmissions()
					time.Sleep(time.Duration(2 * SubmissionTimeout * time.Second))
					fmt.Println("Time up!")
					if len(c.submissions) == 0 {
						fmt.Println("No submissions! :( We'll start next round")
						c.isWinner = true
						c.isJudge = true
					}
					fmt.Println("Which player is the winner?")
					for k, v := range c.submissions {
						fmt.Printf("Player %d submission: %s\n", k, v)
					}

					fmt.Scanf("%d\n", &winnerID)

					if _, ok := c.submissions[winnerID]; ok {
						winnerString := fmt.Sprintf("The winner is Player %d", winnerID)
						fmt.Printf(winnerString)
						c.paxosNode.StartPaxosRoundFor(winnerString) //TODO: What if this fails
						c.isWinner = false
						c.isJudge = false
					}
				} else {
					var submission string

					Submissions := make(chan string)
					kill := make(chan bool, 1)
					fmt.Printf("You have %d seconds to make a submission!\n", SubmissionTimeout)
					go c.AskForSubmission(Submissions, kill)

					select {
					case <-time.After(time.Duration(SubmissionTimeout * time.Second)):
						fmt.Println("Sorry, timeout!")
						kill <- true
					case submission = <-Submissions:
						c.submissions[c.playerID] = submission
						fmt.Println("Submission is: ", submission)
						for {
							succeeded, _ := c.paxosNode.StartPaxosRoundFor("Submission is: " + submission)
							if succeeded == false { // we couldnt make our submission
								time.Sleep(100 * time.Millisecond) // sleep before retrying (should this logic be inside paxos)
							} else {
								break
							}
						}
					}

				}
			}
			if winnerSlot != 0 {
				c.progressing = true
				if winnerID == c.playerID {
					fmt.Printf("Congratulations Player %d, you're the winner!\n", winnerID)
					fmt.Println("************************Pears to Peers **************************")
					c.isWinner = true
					fmt.Println("Lets begin Pears to Peers again with the leader now as ", c.playerID)
					c.submissions = make(map[uint32]string) // make new map for submissions
					c.isJudge = true
					c.judgeID = c.playerID
					fmt.Println("Select a topic")
					var topic string
					n, err := fmt.Scanf("%s\n", &topic)
					if err != nil || n != 1 {
						fmt.Println(n, err)
					}
					topicStmt := fmt.Sprintf("Topic chosen is %s", topic)
					_, err = c.paxosNode.StartPaxosRoundFor(topicStmt) //won't have competing proposals
					if err != nil {
						fmt.Println("Something went wrong!!!", err)
					}
				} else {
					c.isWinner = false
					c.isJudge = false
					c.judgeID = winnerID
					fmt.Printf("Winner is %d\n", winnerID)
				}
				c.submissions = make(map[uint32]string)
			}
			if submissionSlot != 0 {
				c.progressing = true
				c.submissions[msg.ProposerID] = url
			}
		case <-time.After(time.Duration(LeaderTimeout * time.Second)):
			fmt.Println("timed out!!!")
			if !c.progressing {
				timedOut = true
				break
			}
		}

	}
	if !c.progressing {
		c.judgeID = 0
		c.StartGameLoop()
	}
}
