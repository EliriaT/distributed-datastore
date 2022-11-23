package cluster

import (
	"encoding/json"
	"fmt"
	"github.com/EliriaT/distributed-datastore/store"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"
)

var cmInstance *ConsensusModule

const DebugCM = false

type CMState int

const (
	Follower CMState = iota
	Candidate
	Leader
	Dead
)

func (s CMState) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	case Dead:
		return "Dead"
	default:
		panic("unreachable")
	}
}

type LogEntry struct {
	Command DbCommand `json:"command"`
	Term    int       `json:"term"`
}

// ConsensusModule (CM) implements a single node of Raft consensus.
// It embeds a Node struct inside of it
type ConsensusModule struct {
	nodeInstance *Node

	// mu protects concurrent access to a CM.
	mu sync.Mutex

	// Persistent Raft state on all servers
	currentTerm int
	votedFor    int
	LeaderLog   []LogEntry
	myLog       []LogEntry
	peerLogs    [][]LogEntry

	// Volatile Raft state on all servers
	state              CMState
	electionResetEvent time.Time
}

// NewConsensusModule creates a new CM with the current node instance embedded
func GetConsensusModule() *ConsensusModule {
	if cmInstance == nil {
		cmInstance = new(ConsensusModule)
		cmInstance.nodeInstance = GetNode()
		cmInstance.myLog = make([]LogEntry, 0, 100)
		cmInstance.LeaderLog = make([]LogEntry, 0, 100)
		cmInstance.peerLogs = make([][]LogEntry, cmInstance.nodeInstance.numPeers, 10)

		// All the servers first start in the Follower state in Raft protocol,  but in my adaptation, firstly the leader is elected already
		if cmInstance.nodeInstance.IsLeader == true {
			cmInstance.state = Leader
		} else {
			cmInstance.state = Follower
		}

		// all nodes will start the election timer, but the node which is the leader, will see that he is the leader and will stop the for loop
		go func() {

			cmInstance.mu.Lock()
			cmInstance.electionResetEvent = time.Now()
			cmInstance.mu.Unlock()
			cmInstance.runElectionTimer()
		}()
	}

	return cmInstance
}

// electionTimeout generates a pseudo-random election timeout duration. All nodes have different election timeout
func (cm *ConsensusModule) electionTimeout() time.Duration {

	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}

// dlog logs a debugging message if DebugCM > 0.
func (cm *ConsensusModule) dlog(format string, args ...interface{}) {
	if DebugCM == true {
		format = fmt.Sprintf("[%d] ", cm.nodeInstance.Id) + format
		log.Printf(format, args...)
	}

}

func (cm *ConsensusModule) runElectionTimer() {
	// a random election timeout time, each node will have a different election timeout
	timeoutDuration := cm.electionTimeout()
	cm.mu.Lock()
	termStarted := cm.currentTerm
	cm.mu.Unlock()
	cm.dlog("election timer started (%v), term=%d", timeoutDuration, termStarted)

	// This loops until either:
	// - we discover the election timer is no longer needed, (if the node is a leader), or if:
	// - the election timer expires and this CM becomes a candidate; The election timer expires when leader no longer sends heartbeats, and further the election is started
	// In a follower, this typically keeps running in the background for the
	// duration of the CM's lifetime.
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		<-ticker.C

		cm.mu.Lock()
		// if an underlying election does turn the peer into a leader, the concurrent runElectionTimer will just return when observing a state it didn't expect to be in.
		if cm.state != Candidate && cm.state != Follower {
			cm.dlog("in election timer state=%s, bailing out", cm.state)
			cm.mu.Unlock()
			return
		}

		// if a follower gets a RV from a leader in a higher term , this will trigger another becomeFollower call that launches a new electiontimer goroutine. This old election timer will end.
		if termStarted != cm.currentTerm {
			cm.dlog("in election timer term changed from %d to %d, bailing out", termStarted, cm.currentTerm)
			cm.mu.Unlock()
			return
		}

		// Start an election if we haven't heard from a leader or haven't voted for
		// someone for the duration of the timeout.
		if elapsed := time.Since(cm.electionResetEvent); elapsed >= timeoutDuration {
			cm.startElection()
			cm.mu.Unlock()
			return
		}
		cm.mu.Unlock()
	}
}

func (cm *ConsensusModule) startElection() {
	cm.state = Candidate
	cm.currentTerm += 1
	savedCurrentTerm := cm.currentTerm
	cm.electionResetEvent = time.Now()
	// the node will vote for itself
	cm.votedFor = cm.nodeInstance.Id
	cm.dlog("becomes Candidate (currentTerm=%d); log=%v", savedCurrentTerm, cm.myLog)

	votesReceived := 1

	// Send RequestVote to all other peer servers concurrently.
	for _, peerNode := range cm.nodeInstance.Peers {
		go func(peer Node) {
			args := RequestVoteArgs{
				Term:        savedCurrentTerm,
				CandidateId: cm.nodeInstance.Id,
			}
			//var reply RequestVoteReply
			toDoCommand := Command{
				Action:  RequestVote,
				Payload: args,
			}
			byteReq, _ := json.Marshal(toDoCommand)

			cm.dlog("sending RequestVote to %d: %+v", peer.Id, args)
			if reply, err := SendTCPRequest(byteReq, peer.Name+peer.TcpPort); err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()
				cm.dlog("received RequestVoteReply %+v", reply)

				voteReply := convertFromMapToRequestVoteReply(reply.Payload)

				// The node might have won the election because there were enough votes in the other go routines. The response could have arrived slower because of network issues.
				// Or one of the other goroutines from the for loop heard from a server with a higher term, so the node switched back to be a follower.
				// Or if the node heard from a Leader while electing, then it also became a follower in another goroutine
				if cm.state != Candidate {
					cm.dlog("while waiting for reply, state = %v", cm.state)
					return
				}

				// This can happen if another candidate won an election while the node was collecting votes
				if voteReply.Term > savedCurrentTerm {
					cm.dlog("term out of date in RequestVoteReply")
					cm.becomeFollower(voteReply.Term)
					return
					//TODO Question, when the vote would not be granted..?
				} else if voteReply.Term == savedCurrentTerm {
					if voteReply.VoteGranted {
						votesReceived++
						if votesReceived*2 >= cm.nodeInstance.numPeers+1 {
							// Won the election!
							cm.dlog("wins election with %d votes", votesReceived)
							cm.StartLeader()
							return
						}
					}
				}
			}
		}(peerNode)
	}

	//The above for loop is non blocking
	//Run another election timer, in case this election is not successful.
	//This ensures that if nothing useful comes out of this election, a new one will begin after the usual timeout.
	go cm.runElectionTimer()
}

// startLeader switches cm into a leader state and begins process of heartbeats.
// Expects cm.mu to be locked.
func (cm *ConsensusModule) StartLeader() {
	cm.state = Leader
	cm.nodeInstance.IsLeader = true
	cm.dlog("becomes Leader; term=%d, log=%v", cm.currentTerm, cm.myLog)

	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		// Send periodic heartbeats, as long as still leader.
		for {
			cm.leaderSendHeartbeats()
			<-ticker.C

			cm.mu.Lock()
			if cm.state != Leader {
				cm.mu.Unlock()
				return
			}
			cm.mu.Unlock()
		}
	}()

	cm.nodeInstance.SetupRouter()
	go cm.nodeInstance.StartServer()
	go StartWebSocketServer()
}

func (cm *ConsensusModule) leaderSendHeartbeats() {
	//cm.dlog(" started sending heartbeats")
	cm.mu.Lock()
	savedCurrentTerm := cm.currentTerm
	cm.mu.Unlock()

	//The leader will send heartbeats, and receive a response. If in a response it recieves a higher term, it becomes a follower, this means a network partition occurred.
	for _, peer := range cm.nodeInstance.Peers {
		args := AppendEntriesArgs{
			Term:     savedCurrentTerm,
			LeaderId: cm.nodeInstance.Id,
			Entries:  cm.peerLogs[peer.Id-1],
		}

		toDoCommand := Command{
			Action:  AppendEntry,
			Payload: args,
		}
		byteReq, _ := json.Marshal(toDoCommand)

		go func(peer Node) {
			cm.dlog("sending AppendEntries to %v:  args=%+v", peer.Id, args)
			//var reply AppendEntriesReply
			if reply, err := SendTCPRequest(byteReq, peer.Name+peer.TcpPort); err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()

				//reply is of Command type
				appendEntryReply := convertFromMapToAppendEntriesReply(reply.Payload)

				if appendEntryReply.Term > savedCurrentTerm {
					cm.dlog("term out of date in heartbeat reply")
					cm.becomeFollower(appendEntryReply.Term)
					return
				}

				if appendEntryReply.Success == false {
					cm.dlog("leader drags recents logs from follower %v", appendEntryReply.Entries)
					cm.peerLogs[peer.Id-1] = appendEntryReply.Entries
				}
			}
		}(peer)
	}
}

// The election timer is again started
func (cm *ConsensusModule) becomeFollower(term int) {
	cm.dlog("becomes Follower with term=%d; log=%v", term, cm.myLog)
	cm.state = Follower
	cm.nodeInstance.IsLeader = false
	cm.currentTerm = term
	cm.votedFor = -1
	cm.electionResetEvent = time.Now()

	go cm.runElectionTimer()
}

// On the TCP channel commands are received that are either requests for voted or appendEntries from heartbeats or dbCommands
func (cm *ConsensusModule) ListenOnTCP() {

	listen, err := net.Listen("tcp", cm.nodeInstance.Name+cm.nodeInstance.TcpPort)
	if err != nil {
		log.Fatal(err)
	}

	defer listen.Close()
	log.Printf("Node %s TCP server started..", cm.nodeInstance.Name)
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		go cm.handleTCPRequest(conn)
	}
}

func (cm *ConsensusModule) handleTCPRequest(conn net.Conn) {
	var toDoCommand Command

	buffer := make([]byte, 1024)
	length, err := conn.Read(buffer)
	if err != nil {
		log.Fatal(err)
	}

	buffer = buffer[:length]

	err = json.Unmarshal(buffer, &toDoCommand)
	if err != nil {
		log.Fatal(err)
	}

	//log.Printf("Node %d received command %+v", cm.nodeInstance.Id, toDoCommand)
	if toDoCommand.Action == SET {
		var logEntry LogEntry
		cm.mu.Lock()
		logEntry.Term = cm.currentTerm
		cm.mu.Unlock()
		dbCommand := convertFromMapToDbCommand(toDoCommand.Payload)
		logEntry.Command = dbCommand

		store.NodeDataStore.SetValue(dbCommand.Key, []byte(dbCommand.Value))

		cm.myLog = append(cm.myLog, logEntry)
		conn.Write(buffer)
		store.NodeDataStore.PrintStoreContent()
	}
	if toDoCommand.Action == GET {
		dbCommand := convertFromMapToDbCommand(toDoCommand.Payload)
		value, err := store.NodeDataStore.GetValue(dbCommand.Key)
		if err != nil {
			dbCommand.Value = ""
		}
		dbCommand.Value = string(value)
		toDoCommand.Payload = dbCommand
		byteMsg, _ := json.Marshal(toDoCommand)
		conn.Write(byteMsg)
		store.NodeDataStore.PrintStoreContent()
	}
	if toDoCommand.Action == AppendEntry {
		appendEntryArgs := convertFromMapToAppendEntriesArgs(toDoCommand.Payload)
		appendEntryReply, _ := cm.AppendEntries(appendEntryArgs)
		replyCommand := Command{
			Action:  AppendEntry,
			Payload: appendEntryReply,
		}
		byteMsg, _ := json.Marshal(replyCommand)
		conn.Write(byteMsg)
	}
	if toDoCommand.Action == RequestVote {
		requestVoteArgs := convertFromMapToRequestVoteArgs(toDoCommand.Payload)
		requestVoteReply, _ := cm.RequestVote(requestVoteArgs)
		replyVoteCommand := Command{
			Action:  RequestVote,
			Payload: requestVoteReply,
		}
		byteMsg, _ := json.Marshal(replyVoteCommand)
		conn.Write(byteMsg)
	}

	conn.Close()
}

func (cm *ConsensusModule) RequestVote(args RequestVoteArgs) (RequestVoteReply, error) {

	var reply RequestVoteReply
	cm.mu.Lock()
	//log.Println("ajunge dupa lock", args)
	defer cm.mu.Unlock()
	// Not used now
	if cm.state == Dead {
		return reply, nil
	}
	cm.dlog("RequestVote: %+v [currentTerm=%d, votedFor=%d]", args, cm.currentTerm, cm.votedFor)

	if args.Term > cm.currentTerm {
		cm.dlog("... term out of date in RequestVote")
		cm.becomeFollower(args.Term)
	}

	if cm.currentTerm == args.Term &&
		(cm.votedFor == -1 || cm.votedFor == args.CandidateId) {
		reply.VoteGranted = true
		cm.votedFor = args.CandidateId
		cm.electionResetEvent = time.Now()
	} else {
		reply.VoteGranted = false
	}
	reply.Term = cm.currentTerm
	cm.dlog("... RequestVote reply: %+v", reply)
	return reply, nil
}

func (cm *ConsensusModule) AppendEntries(args AppendEntriesArgs) (AppendEntriesReply, error) {
	var reply AppendEntriesReply
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.state == Dead {
		return reply, nil
	}
	cm.dlog("AppendEntries: %+v", args)

	if args.Term > cm.currentTerm {
		cm.dlog("... term out of date in AppendEntries")
		cm.becomeFollower(args.Term)
	}

	reply.Success = false
	if args.Term == cm.currentTerm {
		if cm.state != Follower {
			cm.becomeFollower(args.Term)
		}
		cm.electionResetEvent = time.Now()
		reply.Success = true

		if len(cm.myLog) < len(args.Entries) {
			for i := len(cm.myLog); i < len(args.Entries); i++ {
				cm.ExecuteLogEntry(args.Entries[i])
			}
			//because i allowed a node with less logs to participate in election
			//or a reelected node, has empty logs of individual ones
			// this may happen, but then it should return back the actual logs
		} else if len(cm.myLog) > len(args.Entries) {
			reply.Success = false
			reply.Entries = cm.myLog
			//log.Println("---------Impossible, why this happened.?")
		}

	}

	reply.Term = cm.currentTerm
	cm.dlog("AppendEntries reply: %+v", reply)
	return reply, nil
}

// Only the leader will add log entries. Also it will keep track of other nodes logs
func (cm *ConsensusModule) AddLogEntry(originalNodeId int, replicaNodeId int, command DbCommand) {
	var logEntry LogEntry
	cm.mu.Lock()
	defer cm.mu.Unlock()
	logEntry.Term = cm.currentTerm
	logEntry.Command = command

	cm.LeaderLog = append(cm.LeaderLog, logEntry)

	if cm.nodeInstance.Id == originalNodeId || cm.nodeInstance.Id == replicaNodeId {
		cm.dlog("AppendEntries: %+v", cm.myLog)
		cm.myLog = append(cm.myLog, logEntry)
	}
	if cm.nodeInstance.Id != originalNodeId {
		cm.peerLogs[originalNodeId-1] = append(cm.peerLogs[originalNodeId-1], logEntry)
	}
	if cm.nodeInstance.Id != replicaNodeId {
		cm.peerLogs[replicaNodeId-1] = append(cm.peerLogs[replicaNodeId-1], logEntry)
	}
}

func (cm *ConsensusModule) ExecuteLogEntry(entry LogEntry) {
	dbCommand := entry.Command
	store.NodeDataStore.SetValue(dbCommand.Key, []byte(dbCommand.Value))

	cm.myLog = append(cm.myLog, entry)
}

//TODO USE SOCKETS FOR CM COMMUNICATION BETWEEN THEMSELVES
