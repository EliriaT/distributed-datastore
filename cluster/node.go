package cluster

import (
	"encoding/json"
	"fmt"
	"github.com/EliriaT/distributed-datastore/config"
	"github.com/gorilla/mux"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var nodeInstance *Node

type Node struct {
	Id            int
	Name          string
	Peers         []Node
	numPeers      int
	NodeAddr      string
	HttpPort      string
	router        *mux.Router
	udpPort       string
	broadcastAddr string
	IsLeader      bool
	LeaderName    string
	//IpAddrs       string //maybe will need in future
	TcpPort string
}

type Vote struct {
	Id       int    `json:"id"`
	NodeName string `json:"node_name"`
	NodeAddr string `json:"node_addr"`
	VoteInf  int    `json:"vote"`
	TcpPort  string `json:"tcp_port"`
}

func (n *Node) VoteLeader() Vote {
	return Vote{
		Id:       n.Id,
		NodeName: n.Name,
		NodeAddr: n.NodeAddr,
		VoteInf:  rand.Intn(n.numPeers),
		TcpPort:  n.TcpPort,
	}
}

func (n *Node) SetupRouter() {
	r := mux.NewRouter()
	r.HandleFunc("/get/{key}", GetObject).Methods("GET")
	r.HandleFunc("/set/{key}/{value}", SetObject).Methods("GET")

	n.router = r
}

func (n *Node) StartServer() {
	log.Printf("Node %s HTTP server started..", n.Name)
	log.Fatal(http.ListenAndServe(n.HttpPort, n.router))
}

// func (n *Node) ListenOnTCP() {
//
//		listen, err := net.Listen("tcp", n.Name+n.TcpPort)
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		defer listen.Close()
//		log.Printf("Node %s TCP server started..", n.Name)
//		for {
//			conn, err := listen.Accept()
//			if err != nil {
//				log.Fatal(err)
//				os.Exit(1)
//			}
//			go n.handleTCPRequest(conn)
//		}
//	}
//
//	func (n *Node) handleTCPRequest(conn net.Conn) {
//		var toDoCommand Command
//
//		buffer := make([]byte, 1024)
//		length, err := conn.Read(buffer)
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		buffer = buffer[:length]
//
//		err = json.Unmarshal(buffer, &toDoCommand)
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		log.Printf("Node %d received command %+v", n.Id, toDoCommand)
//		if toDoCommand.Action == SET {
//			dbCommand := convertFromMapToDbCommand(toDoCommand.Payload)
//			store.NodeDataStore.SetValue(dbCommand.Key, []byte(dbCommand.Value))
//			conn.Write(buffer)
//			store.NodeDataStore.PrintStoreContent()
//		}
//		if toDoCommand.Action == GET {
//			dbCommand := convertFromMapToDbCommand(toDoCommand.Payload)
//			value, err := store.NodeDataStore.GetValue(dbCommand.Key)
//			if err != nil {
//				dbCommand.Value = ""
//			}
//			dbCommand.Value = string(value)
//			toDoCommand.Payload = dbCommand
//			byteMsg, _ := json.Marshal(toDoCommand)
//			conn.Write(byteMsg)
//			store.NodeDataStore.PrintStoreContent()
//		}
//		if toDoCommand.Action == AppendEntry {
//
//		}
//		if toDoCommand.Action == RequestVote {
//
//		}
//
//		conn.Close()
//	}
func (n *Node) ListenOnUDP(wg *sync.WaitGroup) {
	var flag = make(chan int, 1500)
	var votes []Vote
	votes = make([]Vote, 0, 3)

	localAddress, _ := net.ResolveUDPAddr("udp", n.udpPort)
	connection, err := net.ListenUDP("udp", localAddress)
	if err != nil {
		log.Fatal(err)
	}
	//defer connection.Close()

	//to try repeat only 3 time the for loop
	go func() {
		for {
			var vote Vote
			buffer := make([]byte, 4096)
			length, addr, err := connection.ReadFromUDP(buffer)
			// error could have happened if the connection was closed and the function restarted
			if err != nil {
				return
			}
			buffer = buffer[:length]

			//err = json.Unmarshal(buffer, &vote)
			//if err != nil {
			//	log.Fatal(err)
			//}

			nodes, vote := unMarshalVoteOrPeersJSON(buffer)
			if nodes == nil {
				fmt.Printf("%s sent this: %+v\n", addr, vote)
				votes = append(votes, vote)
			} else {
				n.Peers = nodes
				flag <- 1
			}

			// check here if i am the leader, then send all the nodes currently available
			// This happens if a peer was disconnected, and reconnected, started voting, but the leader will send all the peers
			if n.IsLeader {
				byteMsg, _ := json.Marshal(n.Peers)
				n.SendBroadCastMessageUDP(byteMsg)
			}

			if len(votes) == n.numPeers {
				flag <- 1
			}
		}
	}()

	<-flag
	if len(n.Peers) > 0 {
		wg.Done()
		return
	}
	countVotes := make([]int, n.numPeers, n.numPeers)
	max := 0
	var leader = -1

	for _, v := range votes {
		countVotes[v.VoteInf]++
		if countVotes[v.VoteInf] > max {
			max = countVotes[v.VoteInf]
		}
	}

	for i := range countVotes {
		if countVotes[i] == max {
			if leader != -1 {
				//This means we have 2 or more nodes with equal votes, and voting sould start again
				log.Printf("Two or more leaders: %d , %d", leader+1, i+1)
				connection.Close()
				n.StartVotingProccess(wg)
				return
			}
			leader = i
		}
	}
	log.Printf("Leader is %d .", leader+1)

	n.GetToKnowPeers(leader, votes)
	wg.Done()

}

func (n *Node) GetToKnowPeers(leader int, votes []Vote) {
	peers := make([]Node, 0, n.numPeers-1)
	myName := n.Name
	leadID := strconv.Itoa(leader + 1)

	if string(myName[len(myName)-1]) == leadID {
		n.IsLeader = true
	}

	for _, v := range votes {
		if v.NodeName != myName {
			node := Node{
				Id:       v.Id,
				Name:     v.NodeName,
				NodeAddr: v.NodeAddr,
				TcpPort:  v.TcpPort,
			}
			peers = append(peers, node)
		}
	}
	n.Peers = peers
}

func (n *Node) SendBroadCastMessageUDP(byteMsg []byte) {
	time.Sleep(time.Duration(rand.Intn(1000)+300) * time.Millisecond)
	broadcastAddress, err := net.ResolveUDPAddr("udp", n.broadcastAddr+n.udpPort)
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Println("aici")
		log.Fatal(err)
	}
	connection, err := net.DialUDP("udp", nil, broadcastAddress)
	if err != nil {
		log.Fatal(err)
	}
	defer connection.Close()

	connection.Write(byteMsg)

}

func (n *Node) StartVotingProccess(wg *sync.WaitGroup) {
	vote := n.VoteLeader()
	byteMsg, _ := json.Marshal(vote)
	go n.ListenOnUDP(wg)
	n.SendBroadCastMessageUDP(byteMsg)
}

func GetNode() *Node {
	//acts like a singleton
	if nodeInstance == nil {
		config.Congif()
		numPeers, _ := strconv.Atoi(config.NodeConfig["num_peers"])
		id, _ := strconv.Atoi(config.NodeConfig["id"])
		nodeInstance = &Node{
			Id:            id,
			Name:          config.NodeConfig["name"],
			NodeAddr:      config.NodeConfig["my_addr"],
			numPeers:      numPeers,
			HttpPort:      config.NodeConfig["http_port"],
			udpPort:       config.NodeConfig["udp_port"],
			broadcastAddr: config.NodeConfig["broadcast_addr"],
			TcpPort:       config.NodeConfig["tcp_port"],
		}
		return nodeInstance
	} else {
		return nodeInstance
	}
}

func getLocalIP() string {
	var localIP string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		os.Stderr.WriteString("Oops: " + err.Error() + "\n")
		os.Exit(1)
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				localIP = ipnet.IP.String()
			}
		}
	}
	return localIP
}
