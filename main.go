package main

import (
	"github.com/EliriaT/distributed-datastore/cluster"
	"log"
	"math/rand"
	"sync"
	"time"
)

func main() {
	log.Println("-----------------------IT IS WORKING--------------------")
	wg := sync.WaitGroup{}
	wg.Add(1)
	rand.Seed(time.Now().UnixNano())

	node := cluster.GetNode()

	node.StartVotingProccess(&wg)
	//Wait for the leader to be elected
	wg.Wait()

	wg.Add(1)
	cm := cluster.NewConsensusModule()
	//if leader start http server, and start heartBeats, else tcp server for replication
	if !node.IsLeader {
		cm.ListenOnTCP()

	} else {
		go cm.StartLeader()
		//node.SetupRouter()
		//node.StartServer()

	}
	wg.Wait()
}

//172.19.255.255
//docker run --name netshoot --rm -it --network cluster nicolaka/netshoot /bin/sh
