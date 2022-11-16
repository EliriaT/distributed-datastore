package cluster

import (
	"fmt"
	"github.com/lithammer/go-jump-consistent-hash"
	"log"
)

const secretSalt = "SM"

var hasher *jump.Hasher

func GetShardAndReplica(key string) (originalShard Node, replicaShard Node) {
	originalKey := key

	hasher = GetHasher()
	originalID := hasher.Hash(key) + 1
	replicaID := hasher.Hash(key+secretSalt) + 1

	for replicaID == originalID {
		key = key + secretSalt
		replicaID = hasher.Hash(key) + 1
	}

	log.Printf("For key %s, the original and replica nodes are: %d, %d", originalKey, originalID, replicaID)

	originalShard, err := SearchForPeerById(originalID)
	//log.Println("Id is : ", originalShard)
	if err != nil {
		log.Fatal("Could not find node with such id")
	}
	replicaShard, err = SearchForPeerById(replicaID)
	//log.Println("Id is : ", replicaShard)
	if err != nil {
		log.Fatal("Could not find node with such id")
	}
	return originalShard, replicaShard

}

func GetHasher() *jump.Hasher {
	if hasher == nil {
		node := GetNode()
		hasher = jump.New(node.numPeers, jump.NewCRC64())
		return hasher
	}
	return hasher
}

func SearchForPeerById(id int) (Node, error) {
	node := GetNode()

	if node.Id == id {
		return *node, nil
	}
	for _, p := range node.Peers {
		if p.Id == id {
			return p, nil
		}
	}
	return Node{}, fmt.Errorf("Node not found")
}
