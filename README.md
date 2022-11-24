# distributed-datastore

Extremely simple distributed in-memory key-value datastore written in Go, that uses the Raft Consensus Algorithm for solving fault-tolerance problems in  distributed systems and Jump consistent hash algorithm for evenly dividing the key space among the `n` buckets and of evenly dividing the workload when the number of buckets changes.

1. To run the app in a docker container, first build the image:

`docker build -t node .`

2. Then use docker compose:

`docker compose up`

