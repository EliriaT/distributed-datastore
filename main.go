package main

import (
	"github.com/EliriaT/distributed-datastore/cluster"
	"math/rand"
	"sync"
	"time"
)

func main() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	rand.Seed(time.Now().UnixNano())

	node := cluster.GetNode()

	node.StartVotingProccess(&wg)
	//Wait for the leader to be elected
	wg.Wait()

	//if leader start http server, else tcp server for replication
	if node.IsLeader {
		node.SetupRouter()
		node.StartServer()
	} else {
		node.ListenOnTCP()
	}

}

//172.19.255.255
