Pears to Peers - paxos based replicated log that helps you play your favorite game, with support for quiescing as well!!

Description:
our version of “Pears to Peers” is a variation of the standard “Apples to Apples” card game. The game requires at least 3 players and proceeds as follows: In the first round a judge is chosen who then picks a topic that will be the subject of that round. Then all players must submit url for content that illustrate that subject in some way. The leader then picks a winner out of all the submissions. The winner earns a point and becomes the new leader for the next round.

The application is a peer-to-peer multiplayer game. There is one central server that handles organizing players into groups before they start a game. The central server keeps a record of the game: players involved, etc. However, once a game begins, the game state is entirely maintained by peer-to-peer communication between members of the game. Note that the updates to game state are not restricted to come only via the leader (since each players submissions are broadcasted), and the application implements features that allow the game to proceed with a newly selected judge even if the original judge fails.
Each player has a game client that is used to make submissions. The app uses a Paxos replicated log to guarantee that the individual game clients agree on the game state in different critical stages of the game. 

Features implemented:
paxos based features:
1) paxos based replicated log
2) game continues till f failures
3) can tolerate network delays
4) paxos is used to deal with multiple players wanting to be judge
5) support for quiescing and gettting back a dead node

other game features:
1) can ensure judge failover

Tests:
We have the following tests for our paxos implementation:
All of the tests ensure that the log is consistent at the end of the test
1) NewPaxosNode - creates a new paxos node
2) OnePaxosRound - runs one round of paxos
3) MultiplePaxosRounds(num_rounds) - runs num_rounds rounds of paxos
4) EveryoneProposes - makes multiple people compete to be leader of the paxos round.
5) DelayNodes - introduces network delays in the paxos round. It does so by passing in a delay parameter to the paxos node. The node simply sleeps for that time before doing any activity. In normal operation, this is 0
6) DelayNodes2 - two different sets of delays are introduced
7) FailureLimit - tests that the app can tolerate upto f failures

To run the app:
1) start the central server using ./src/github.com/pears-to-peers/scripts/start_central_server.sh
2) start the clients using go run src/github.com/pears-to-peers/runners/cli/cli.go -port=8001
3) you will need minimum 3 players to start
4) Enter s to start
5) You will be prompted to ask if you want to be judge
6) If yes, you are judge. Then you choose topic and wait for submissions. After that you pick a winner
If you are not judge, you wait for topic to be chosen and then make submissions. If you are winner, you become judge for the next round. 

To start quiescing:
1) First of all enter "Y" on the central server prompt of "Do you want to quiesce a node"
2) It will ask you for node id and hostport of the deadnode.
3) Then it sends out messages to the other players to start quiesce.
4) During this
	a) Nodes dont accept new messages
	b) start rounds for uncommitted slots with nil values
	c) flush any pending messages
5) The log is copied over to the dead node.
6) The dead node is restarted with
go run src/github.com/pears-to-peers/runners/cli/cli.go -quiescing=true -cluster=/tmp/clusterInfo -nodeID=2 -port=8002

The cluster parameter is used by the client to know its cluster. The file basicaly contains lines like this:
{"HostPort":":8001","NodeID":1}
{"HostPort":":8003","NodeID":3}
{"HostPort":":8002","NodeID":2}

Once this is done, the dead node participates normally with the rest of the nodes.
The code for quiescing can be found in:
a) src/github.com/pears-to-peers/centralServer/central_server_impl.go
b) src/github.com/pears-to-peers/cli/cli_impl.go
c) src/github.com/pears-to-peers/paxos/paxosnode/paxosnode.go

To run the test:
go run src/github.com/pears-to-peers/paxos/tests/paxosTest.go

Some architectural decisions:
a) Log is used to decide what to do next in the game. That is, the log messages are read one after the other to decide what to do next. As a result, as long as the log is consistent, players always know whats happening in the game and can catch up
b) If multiple people want to be judge (not paxos proposer), their messages are written to log, and the first judge entry that goes into the log is the player who becomes judge.
Eg. if log contains
"A wants to be judge"
"B wants to be judge"
then A is judge. Note both the above values are successfully commiitted paxos round values in the log.
c) paxos nodes have in-memory slots that replicate the log state in memory. They contain important information like where a particular slot info is in the log file, whether the slot is committed or not.
