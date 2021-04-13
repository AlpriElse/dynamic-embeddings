# Simple Distributed File System
### Overview

**Features**

* Master fault tolerance with leader election
* Strong file consistency
* Auto-replication during node failures
* Distributed lock management

**Performance requirements**

* Fault tolerant up to **> 3 node failures**

### Usage

To compile and run:

```
go build -o main
./main
```

Configure introducer, port, and intervals in config.json

```json
{
    "service": {
        "failure_detector": "gossip",
        "introducer_ip": "172.22.156.42",
        "port": 9090,
        "initial_master_ip": "172.22.156.42",
        "master_port": 9091,
        "file_port": 9092
    },
    "settings": {
        "gossip_interval": 1,
        "all_interval": 3,
        "fail_timeout": 5,
        "cleanup_timeout": 24,
        "num_processes_to_gossip": 3,
        "replication_factor": 4
    }
}
```

CLI

```
-- Maple Juice --
maple <maple_exe> <num_maples> <sdfs_intermediate_filename_prefix> <sdfs_src_directory>                    - maple
juice <juice_exe> <num_juices> <sdfs_intermediate_filename_prefix> <sdfs_dest_filename> delete_input={0,1} - juice

-- SDFS --
put <local file path> <remote filename> - upload file
get <local file path> <remote filename> - download file
delete <remote filename>				- delete file
ls <filename>							- list ips containing file
store									- list file stored in node

-- Membership --
join									- join the group and start heartbeating
join introducer							- create a group as the introducer
leave									- leave the group and stop heartbeating
status									- get live status of group
whoami									- get self id
master									- return id of master
switch <protocol>						- switch system protocol between options of: 

-- Logger --
get logs								- pull the current local log
grep 									- grep for logs from other group members
stop									- manually stop heartbeating
kill									- exit gracefully
switch <protocol>						- switch system protocol between options of: 											gossip, alltoall

-- Metrics --
metrics									- get current failure/bandwidth stats
sim <test>								- debug only; simulation between options of: 											failtest
```



## Design

### Components

* File transfer service
* Leader election
* File replication recovery

Directory structure

```
root
	config.json
	src
		main.go			// main program
		net.go			// networking library
		util.go			// utilities (config, etc)
		detector.go		// gossip/all-to-all handlers
		monitor.go
		logs.go
```

### Leader Election

Leader election is implemented with the bully algorithm. Initially, the master runs as the introducer. A node can call for an election when it detects it's master has failed. When a node calls for election, it sends a message to nodes with higher ids than itself whether or not they are alive. If at least one node with a higher id responds, it will acknowledge the message with an OkMsg. The node that sends the acknowledgement will run it's own election, sending the same message to nodes with higher ids. Eventually, a node that does not receive an acknowledgement will conclude that it has the highest ID among living nodes. It will send a CoordinatorMsg to all nodes with lower IDs that it has declared itself master.

### Replication

SDFS is tolerant to three machine failures. When a file is first requested to be uploaded, the master ensures that it is uploaded to four nodes. Whenever the current master detects a failure of a node, it goes over a map containing all the files stored by that node. For each such file, it requests an alive node containing the file to upload it to another node which doesn’t contain the file - this is retried until either all replicas are dead (which can’t happen since we’re limited to three failures), or no alive node is available to create an additional replica (which won’t happen once we have at least seven machines, in case of three failures. In this way, it is always ensured that four replicas of a file are always alive.

### Logging

* Use Info.Println("message"), Warn.Println("message"), or Err.Println("message")

