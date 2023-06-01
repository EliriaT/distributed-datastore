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
 * A Follower will grant vote for a Candidate requestVote, if his current term is the same as the Candidate's term.
 * The Leader keeps track of the logs of each peer node.
 * The Leader sends heartbeats to each Follower. A heartbeat contains the current Term, leader id, and  the logs of the Follower.
 * The Follower sends a heartbeat response. A heartbeat response contains the Follower's Term, a boolean succes value, the Follower's logs.
 * If a Leader detects a heartbeat response from a node with a higher Term, it switches to Follower state. This means a network partition happened.
 * If the boolean success field in the heartbeat response is False, it means logs[nodeID] of the Leader and Candidate does not correspond, and Follower has more logs. In this situation, the Leader will copy the response logs, to its logs[nodeID]
 * If the Follower sees that his logs does not correspond, Leader has more logs, the Follower will copy these logs to itself.
 * Each node is a TCP server 
 * Followers receive on TCP AppendEntry commands which act as a heartbeat
 * Leader receives AppendEntryReply, which is the hearbeat reply
 * Leader represents a HTTP server listening for GET and SET commands.
 * Websocket connection to Leader is allowed,  in order to view real time updates of the storage.
 * Replication of data on 2 different nodes.
 
# Project-Lab Requirements

Develop distributed in memory datastore which could be used as a web service. Datastore have to be distributed, this means you should have at least 3 servers which stores the data. In memory data
store represents just a store which holds data as cache without storing it in any dedicated storage such as SQL or NoSQL database or any other storage. Your data store has to be used as a web service, this means all data operations have to be performed as network requests.
Datastore has to be able to perform CRUD (Create, Read, Update, Delete) operations on data you provide.

The datastore is storing the data in distributed way, which means the data you provide could be stored in any of datastore
partition cluster server. The idea here is to have multiple servers which stores provided data without exposing to the client how
and where data is actually stored. Accessing the data from datastore should return requested data in case it is store on any other partition cluster servers.

Client sends a network request to the datastore partition leader(explained below) and
data is operated on the server where request is handled or could be operated on any other server in case the initial server
decides to forward current data operation to any other server from partition cluster. The response to the user for any request is
provided in the same way the data was operated on the initial server.

It is your responsibility to handle any data distribution, sharing, synchronization, integrity and operations in the whole cluster.

The cluster includes at least 3 servers which stores the data. One of the cluster server will play role of Partion Leader . At cluster start up, all servers will synchronize between and the cluster itself will
decide who is partition leader and will provide the dedicated output message. The clients will communicate only with Partion
Leader , but under the hood it manages data distribution and load balancing of the whole cluster by communicating with other
cluster servers.

Cluster itself represents the datastore and each server is just a part of this distributed datastore. In oder to keep the datastore
fault tolerant, data have to be duplicated on multiple servers in such a way that even in case when half(50%) of the servers will
be unavailable(shutdown), the datastore still will be capable for providing all the data, previously stored.

Periodically servers have to synchronize the data they storing in order to avoid any data corruptions as the same data could be
duplicated on multiple servers.Cluster should be started and managed as a set of docker containers which hosts replica of the same application.
Cluster servers have to communicate with each other using "low level" protocols, such as TCP and/or UDP. It is up to you to
decide which protocol for which scenario to use. You can use both TCP and UDP or just one of them. 

Clients have to communicate with your datastore using higher level protocols. The main protocol to interact with datastore
should be HTTP(S). For more advanced use cases consider to use FTP for storing files and WebSockets for real time data
updates.
