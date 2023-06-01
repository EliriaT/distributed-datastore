# distributed-datastore

A distributed in-memory key-value datastore written in Go, inspired and based on the Raft Consensus Algorithm for solving fault-tolerance problems in  distributed systems and on Jump consistent hash algorithm for evenly dividing the key space among the `n` buckets and of evenly dividing the workload when the number of buckets changes.

1. To run the app in a docker container, first build the image:

`docker build -t node .`

2. Then use docker compose:

`docker compose up`

# Implemented:
 * Leader ellection on application start and on leader disconection / failure
 * Each node has three states: Follower, Candidate, Leader, Dead
 * The leader sends ocasional heartbeats, to signal that it is alive
 * In a Follower node, when the election timer expires because no heartbeat is received from leader, the node becomes a Candidate
 * Each node has logs that allow leader's state replication, in order to guaranteer data synchronization.
 * When a node becomes a candidate, it starts requesting votes from peers
 * Nodes do not become candidates simultaneously in the exact same time, but multiple elections can run simultaneously. Only one leader allowed.
 * Each node has a Term. The Term increases on each election start.
 * Election in a node is cancelled, and it becomes a Follower if: hearing a heartbeat from another Leader, or a request vote from a node with a higher Term
 * A node becomes Leader as a result of election, if half of peer nodes voted for this Candidate
 * The Leader keeps track of the logs of each peer node.
 * The Leader sends heartbeats to each Follower. A heartbeat contains the current Term, leader id, and  the logs of the Follower.
 * The Follower sends a heartbeat response. A heartbeat response contains the Follower's Term, a boolean succes value, the Follower's logs.
 * If a Leader detects a heartbeat response from a node with a higher Term, it switches to Follower state. This means a network partition happened.
 * If the boolean success field in the heartbeat response is False, it means logs[nodeID] of the Leader and Candidate does not correspond. In this situation, the Leader will copy the response logs, to its logs[nodeID]
 * 
# Project-Lab Requirements
