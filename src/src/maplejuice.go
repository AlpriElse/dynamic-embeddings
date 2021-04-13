package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MapleJuice structs
type MapleJuiceQueueRequest struct {
	RequestingId       uint8
	IsMaple            bool
	FileList           []string
	ExeName            string
	NumTasks           int
	IntermediatePrefix string
	DeleteInput        bool
}

type MapleJuiceReply struct {
	Completed bool
	KeyList   []string
}

// Maple structs
type MapleRequest struct {
	ExeName            string
	IntermediatePrefix string
	FileName           string
	BlockNum           int
}

type Task struct {
	Request  MapleRequest
	Replicas []net.IP
}

// Juice structs
type JuiceRequest struct {
	ExeName            string
	IntermediatePrefix string
	Key                string
	PartitionId        int
}

type JuiceTask struct {
	Request JuiceRequest
	Nodes   []net.IP
}

type KeyLocationRequest struct {
	Key string
}

type KeyLocationReply struct {
	Nodes []net.IP
}

// Status structs
type Status int

const (
	None Status = iota
	RequestingMaple
	MapleOngoing
	MapleFinished
	RequestingJuice
	JuiceOngoing
)

// Directories

const (
	juiceTempDir = "juiceTemp"
)

var (
	mapleJuiceCh = make(chan Status, 1)
	queue        []MapleJuiceQueueRequest
	currTasks    = make(map[string]Task) // fileName -> [replicaIPs]
	taskLock     sync.Mutex
	lastStatus   Status = None
)

// (master) listens for changes in the maple/juice process for the task run queue
func (node *SdfsNode) ListenMapleJuice() {
	for {
		// blocks until there is a change in the status
		switch status := <-mapleJuiceCh; status {
		case None:
			go node.RunFirstMaple()

		case RequestingMaple:
			if lastStatus == None {
				go node.RunFirstMaple()
			}

		case MapleFinished:
			lastStatus = MapleFinished
			go node.RunFirstJuice()

		case RequestingJuice:
			if lastStatus == MapleFinished {
				go node.RunFirstJuice()
			}

		default:
			break
		}
	}
}

// (master) loop through the queue and initiate maple on the first maple task
func (node *SdfsNode) RunFirstMaple() {
	// initiate maple
	for idx := 0; idx < len(queue); idx += 1 {
		task := queue[idx]
		if task.IsMaple {
			node.Maple(task)
			node.RemoveFromQueue(idx)
			break
		}
	}
}

// (master) loop through the queue and initiate juice on the first maple task
func (node *SdfsNode) RunFirstJuice() {
	for idx := 0; idx < len(queue); idx += 1 {
		task := queue[idx]
		if !task.IsMaple {
			node.Juice(task)
			node.RemoveFromQueue(idx)
			break
		}
	}
}

// (master) remove task from queue
func (node *SdfsNode) RemoveFromQueue(idx int) {
	queue = append(queue[:idx], queue[idx+1:]...)
}

// (worker) call master to add task to its queue
func (node *SdfsNode) QueueTask(mapleQueueReq MapleJuiceQueueRequest) error {
	var res MapleJuiceReply
	return client.Call("SdfsNode.AddToQueue", mapleQueueReq, &res)
}

// (master) add task to blocking queue and fill channel to notify the listener
func (node *SdfsNode) AddToQueue(mapleQueueReq MapleJuiceQueueRequest, reply *MapleJuiceReply) error {
	queue = append(queue, mapleQueueReq)

	if mapleQueueReq.IsMaple {
		mapleJuiceCh <- RequestingMaple
	} else {
		mapleJuiceCh <- RequestingJuice
	}

	return nil
}

// (master) prompts worker machines to run maple on their uploaded blocks
func (node *SdfsNode) Maple(mapleQueueReq MapleJuiceQueueRequest) {
	lastStatus = MapleOngoing
	startTime := time.Now()
	Info.Println("Beginning Map phase.")

	chanSize := 100
	mapleCh := make(chan Task, chanSize)

	for _, localFName := range mapleQueueReq.FileList {
		sdfsFName := node.Master.sdfsFNameMap[localFName]
		if blockMap, ok := node.Master.fileMap[sdfsFName]; ok {
			// initiate maple on each block of each file
			var req MapleRequest
			req.ExeName = mapleQueueReq.ExeName
			req.IntermediatePrefix = mapleQueueReq.IntermediatePrefix
			req.FileName = sdfsDirName + "/" + sdfsFName

			numBlocks := node.Master.numBlocks[sdfsFName]
			for i := 0; i < numBlocks; i++ {
				req.BlockNum = i
				ips := blockMap[i]
				if len(ips) > 0 {
					mapleCh <- Task{req, ips}
				}

			}
		}
	}
	node.RunTasks(mapleCh, mapleQueueReq.NumTasks)
	duration := time.Since(startTime)
	node.SendMessage(mapleQueueReq.RequestingId, "Finished Maple")
	node.SendMessage(mapleQueueReq.RequestingId, "Elapsed: "+strconv.FormatFloat(float64(duration)/1000000000, 'f', 3, 64)+" s")
}

// (master) run the tasks in the job queue on NumMaples/NumJuices # of tasks
func (node *SdfsNode) RunTasks(tasks chan Task, numTasks int) {
	var wg sync.WaitGroup

	Info.Println("RunTasks entered")
	// initialize workers
	for i := 0; i < numTasks; i++ {
		go node.RunTaskWorker(i, &wg, tasks)
		wg.Add(1)
	}

	// stopping the worker and waiting for them to complete
	close(tasks)
	wg.Wait()

	Info.Println("Completed Maple phase.")
	mapleJuiceCh <- MapleFinished
}

// (master) worker reads from job queue and initializes maple on the current job
func (node *SdfsNode) RunTaskWorker(i int, wg *sync.WaitGroup, tasks <-chan Task) {
	for task := range tasks {
		wg.Add(1)

		// Find chosenIp or whatever
		// Call RPC.Maple/Juice function here (RequestMapleOnBlock)
		taskLock.Lock()
		currTasks[task.Request.FileName+".blk_"+fmt.Sprint(task.Request.BlockNum)] = task
		taskLock.Unlock()
		err := node.RequestMapleOnBlock(task.Replicas[0], task.Request)

		for err != nil {
			// keep trying until success or you run out of options
			err = node.RescheduleTask(task.Request.FileName + ".blk_" + fmt.Sprint(task.Request.BlockNum))
		}

		taskLock.Lock()
		delete(currTasks, task.Request.FileName)
		taskLock.Unlock()
		wg.Done()
	}

	wg.Done()
}

// (master) makes rpc call to worker machine to run maple on a specified file block
func (node *SdfsNode) RequestMapleOnBlock(chosenIp net.IP, req MapleRequest) error {
	mapleClient, err := rpc.DialHTTP("tcp", chosenIp.String()+":"+fmt.Sprint(Configuration.Service.masterPort))
	if err != nil {
		fmt.Println("Error in connecting to maple client ", err)
		return err
	}

	var res MapleJuiceReply
	err = mapleClient.Call("SdfsNode.RpcMaple", req, &res)
	if err != nil || !res.Completed {
		fmt.Println("Error: ", err, "res.completed = ", res.Completed)
	} else {
		if _, ok := node.Master.prefixKeyMap[req.IntermediatePrefix]; !ok {
			node.Master.prefixKeyMap[req.IntermediatePrefix] = make(map[string]bool)
		}
		for _, key := range res.KeyList {
			prefixKey := req.IntermediatePrefix + "_" + key
			if checkMember(chosenIp, node.Master.keyLocations[prefixKey]) == -1 {
				node.Master.keyLocations[prefixKey] = append(node.Master.keyLocations[prefixKey], chosenIp)
			}
			if _, ok := node.Master.prefixKeyMap[req.IntermediatePrefix][key]; !ok {
				node.Master.prefixKeyMap[req.IntermediatePrefix][key] = true
			}
		}
	}

	return err
}

// (master) reschedule task to another machine that has that file
// 			initiated when a worker has failed
func (node *SdfsNode) RescheduleTask(fileName string) error {
	Info.Println("Rescheduling task ", fileName)
	taskLock.Lock()
	if task, ok := currTasks[fileName]; ok {
		replicas := task.Replicas
		if len(replicas) < 2 {
			// no more to try
			// TODO: re-replicate file elsewhere, and try again?
			//		for now, delete + ignore lol
			Err.Println("Could not successfully finish task on ", fileName)
		} else {
			newReplicas := make([]net.IP, len(replicas)-1)
			copy(newReplicas, replicas[1:])
			currTasks[fileName] = Task{task.Request, newReplicas}

			taskLock.Unlock()
			return node.RequestMapleOnBlock(newReplicas[0], task.Request)
		}
	}

	taskLock.Unlock()
	return nil
}

// (worker) receives Request to run a maple_exe on some file block from the master
func (node *SdfsNode) RpcMaple(req MapleRequest, reply *MapleJuiceReply) error {
	// format: fileName.blk_#
	blockNum := strconv.Itoa(req.BlockNum)
	filePath := req.FileName + ".blk_" + blockNum

	Info.Println("Executing maple on ", filePath)

	var response MapleJuiceReply

	arg0 := "./" + req.ExeName
	arg1 := filePath

	cmd := exec.Command(arg0, arg1)
	Info.Println("Executing ", arg0, arg1)
	fmt.Println("Executing ", arg0, arg1)

	output, err := cmd.Output()

	if err != nil {
		fmt.Println("Error in executing maple.")
		response.Completed = false
	} else {
		response = WriteMapleKeys(string(output), req.IntermediatePrefix)
	}

	fmt.Print("> ")

	*reply = response
	return err
}

// (worker) Scan output of maple in the format [key,key's value] and store to intermediate files by key
func WriteMapleKeys(output string, prefix string) MapleJuiceReply {
	// read output one line at a time
	scanner := bufio.NewScanner(strings.NewReader(output))
	keySet := make(map[string]bool)
	for scanner.Scan() {
		commaSplitter := func(c rune) bool {
			return c == ','
		}
		keyVal := strings.FieldsFunc(scanner.Text(), commaSplitter)
		if len(keyVal) < 2 {
			continue
		}
		key := keyVal[0]
		val := keyVal[1]

		// TODO: convert key to an appropriate string for a file name
		keyString := key
		prefixKey := prefix + "_" + keyString

		// write to MapleJuice/prefix_key
		// key -> ip
		// need method to get all keys
		filePath := path.Join([]string{mapleJuiceDirName, prefixKey}...)
		f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)

		if err != nil {
			fmt.Println("Error opening ", filePath, ". Error: ", err)
			continue
		}

		if _, err := f.WriteString(val + "\n"); err != nil {
			fmt.Println("Error writing val [", val, "] to ", filePath, ". Error: ", err)
		}
		keySet[key] = true
		f.Close()
	}
	keyList := make([]string, 0, len(keySet))
	for k := range keySet {
		keyList = append(keyList, k)
	}
	return MapleJuiceReply{Completed: true, KeyList: keyList}
}

// locally grab files in the directory
func GetFileNames(dirName string) []string {
	var fileNames []string

	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		fmt.Println("Error: ", dirName, " is not a valid directory.")
	} else {
		for _, f := range files {
			fileNames = append(fileNames, f.Name())
		}
	}

	return fileNames
}

/*
   Juice features
*/
// (master) prompts worker machines to run juice on their uploaded blocks
func (node *SdfsNode) Juice(juiceQueueReq MapleJuiceQueueRequest) {
	// Update status
	lastStatus = JuiceOngoing
	startTime := time.Now()
	node.SendMessage(juiceQueueReq.RequestingId, "Beginning Juice phase.")
	fmt.Print("> ")

	// Get keys of given prefix
	keysMap := node.Master.prefixKeyMap[juiceQueueReq.IntermediatePrefix]
	// Mapkeys to key list
	keys := make([]string, 0)
	for k, _ := range keysMap {
		keys = append(keys, k)
	}
	// Run partitioner at juice request
	numJuices := juiceQueueReq.NumTasks
	partitions := partitioner(keys, numJuices, false)
	outputFname := juiceQueueReq.FileList[0]

	// Create job scheduling structs
	juiceCh := make(chan JuiceTask, len(keys))
	var wg sync.WaitGroup

	// Start workers
	node.RunJuiceWorkers(&wg, juiceCh, juiceQueueReq.NumTasks)

	// Request a juice task
	for id, keyList := range partitions {
		// Get ip list, choose node with key id % numNodes
		for _, key := range keyList {
			var req JuiceRequest
			req.ExeName = juiceQueueReq.ExeName
			req.IntermediatePrefix = juiceQueueReq.IntermediatePrefix
			req.Key = key
			req.PartitionId = id
			// Get ip address of id
			nodeId, _ := node.FindAvailableNode(req.PartitionId)
			chosenIp := []net.IP{node.Member.membershipList[nodeId].IPaddr}
			// Send Juice Request to that partition, with key to reduce
			wg.Add(1)
			juiceCh <- JuiceTask{req, chosenIp}
		}
	}
	// Wait for all workers/tasks to complete
	wg.Wait()
	close(juiceCh)
	Info.Println("All juice tasks complete")
	// Tasks complete, create a new file with all juice outputs
	node.CollectJuices(juiceQueueReq.IntermediatePrefix, keys, outputFname)
	// Upload file to SDFS
	sessionId := node.RpcLock(int32(node.Member.memberID), outputFname, SdfsLock)
	node.RpcPut(outputFname, outputFname)
	_ = node.RpcUnlock(sessionId, outputFname, SdfsLock)
	// Remove output file from master
	_ = os.Remove(outputFname)
	// indicate when it's done
	Info.Println("Completed Juice phase.")
	Info.Print("> ")
	duration := time.Since(startTime)
	node.SendMessage(juiceQueueReq.RequestingId, "Finished Juice.")
	node.SendMessage(juiceQueueReq.RequestingId, "Elapsed: "+strconv.FormatFloat(float64(duration)/1000000000, 'f', 3, 64)+" s")
	// Remove input if needed
	if juiceQueueReq.DeleteInput {
		delete(node.Master.prefixKeyMap, juiceQueueReq.IntermediatePrefix)
		for _, k := range keys {
			delete(node.Master.keyLocations, juiceQueueReq.IntermediatePrefix+"_"+k)
		}
	}
	node.SendMessage(juiceQueueReq.RequestingId, "Deleted inputs.")

	lastStatus = None
	mapleJuiceCh <- None
}

// (master) makes rpc call to worker machine to run maple on specific file block
func (node *SdfsNode) RequestJuiceTask(chosenIp net.IP, req JuiceRequest) error {
	mapleClient, err := rpc.DialHTTP("tcp", chosenIp.String()+":"+fmt.Sprint(Configuration.Service.masterPort))
	// Call RpcMaple at chosen IP
	if err != nil {
		Info.Println(err)
	}

	var res MapleJuiceReply
	err = mapleClient.Call("SdfsNode.RpcJuice", req, &res)
	if err != nil || !res.Completed {
		// Reschedule juicer by sending error
		Info.Println("Error: ", err, "res.completed = ", res.Completed)
		return err
	}
	// Complete juice and check if all juices finished
	return nil
}

// (master) Start numTasks # of juice workers
func (node *SdfsNode) RunJuiceWorkers(wg *sync.WaitGroup, tasks chan JuiceTask, numTasks int) {
	for workerId := 0; workerId < numTasks; workerId++ {
		go node.RunJuiceWorker(workerId, wg, tasks)
	}
}

// (master) Reschedule a juice task
func (node *SdfsNode) RescheduleJuiceTask(wg *sync.WaitGroup, task JuiceTask, tasks chan JuiceTask) {
	wg.Add(1)
	// Get ip address of id
	nodeId, _ := node.FindAvailableNode(task.Request.PartitionId)
	chosenIp := []net.IP{node.Member.membershipList[nodeId].IPaddr}
	tasks <- JuiceTask{task.Request, chosenIp}
	wg.Done()
}

// (master) Run a juice worker
func (node *SdfsNode) RunJuiceWorker(id int, wg *sync.WaitGroup, tasks chan JuiceTask) {
	for task := range tasks {
		// Request juice task on a worker
		err := node.RequestJuiceTask(task.Nodes[0], task.Request)
		if err != nil {
			wg.Add(1)
			node.RescheduleJuiceTask(wg, task, tasks)
		} else {
			// Download juice output from corresponding file path of key + prefix
			Info.Println("Getting juice output from worker", id)
			prefixKey := task.Request.IntermediatePrefix + "_" + task.Request.Key
			juiceFilePath := filepath.Join(juiceTempDir, prefixKey)
			err = Download(task.Nodes[0].String(), fmt.Sprint(Configuration.Service.filePort), juiceFilePath, juiceFilePath)
			if err != nil {
				fmt.Println("Error retrieving juice output from worker ", id, " | ", err)
			} else {
				fmt.Println("Succesfully retrieved from worker", id)
			}
		}
		wg.Done()
	}
}

// (master) collect all juice after all tasks completed
func (node *SdfsNode) CollectJuices(prefix string, keys []string, outFname string) {
	Info.Println("Collecting all juices for", keys)
	Info.Println("Saving juices to", outFname)
	// Save all juices to local folder of collected juice
	fileFlags := os.O_CREATE | os.O_WRONLY
	file, err := os.OpenFile(outFname, fileFlags, 0777)
	if err != nil {
		Info.Println(err)
	}
	defer file.Close()
	juices := []byte{}

	for _, key := range keys {
		juiceFilePath := filepath.Join(juiceTempDir, prefix+"_"+key)
		Info.Println("Collecting", juiceFilePath)

		bytes, err := ioutil.ReadFile(juiceFilePath)
		if err != nil {
			Info.Println(err)
		}
		juices = append(juices, bytes...)
	}
	_, err = file.Write(juices)
	if err != nil {
		Info.Println(err)
	}

	Info.Println("Finished collecting juice in", outFname)
}

// (master) Helper function to find IP
func (node *SdfsNode) FindAvailableNode(id int) (uint8, error) {
	id = id % len(node.Member.membershipList)
	for _, currNode := range node.Member.membershipList {
		if id == 0 {
			return currNode.MemberID, nil
		}
		id -= 1
	}
	return 0, errors.New("Cannot find node")
}

// (worker) Runs on node assigned to task
func (node *SdfsNode) RpcJuice(req JuiceRequest, reply *MapleJuiceReply) error {
	prefix := req.IntermediatePrefix
	key := req.Key
	exeName := "./" + req.ExeName
	// Run shuffler
	sortedFruits := node.ShuffleSort(prefix, key)
	// Execute juicer on key
	err := ExecuteJuice(exeName, prefix, key, sortedFruits)
	// Set completed
	var resp MapleJuiceReply
	resp.Completed = true
	*reply = resp
	return err
}

// (master) GetKeyLocations
func (node *SdfsNode) GetKeyLocations(req KeyLocationRequest, reply *KeyLocationReply) error {
	ipList, ok := node.Master.keyLocations[req.Key]
	if !ok {
		return errors.New("No prefix key, " + req.Key + ",exists")
	}
	var resp KeyLocationReply
	resp.Nodes = ipList
	*reply = resp
	return nil
}

// (worker) RpcGetKeyLocations
func (node *SdfsNode) RpcGetKeyLocations(key string) ([]net.IP, error) {
	req := KeyLocationRequest{Key: key}
	var res KeyLocationReply
	err := client.Call("SdfsNode.GetKeyLocations", req, &res)
	if err != nil {
		Info.Println("Failed:", err)
		return []net.IP{}, errors.New("get key locations failed")
	}
	return res.Nodes, nil
}

// (worker) Pull data and shuffle/sort into a single value
func (node *SdfsNode) ShuffleSort(prefix string, key string) []byte {
	prefixKey := prefix + "_" + key
	Info.Println("Startng shuffle sort on", prefixKey)
	// Get ips of file
	ipList, err := node.RpcGetKeyLocations(prefixKey)
	if err != nil {
		return []byte{}
	}
	sorted := make([]byte, 0)
	juiceTempPath := filepath.Join(juiceTempDir, prefixKey)
	filePath := filepath.Join(mapleJuiceDirName, prefixKey)
	for _, ipAddr := range ipList {
		Info.Println("Downloading from ", filePath, " to", juiceTempPath, " from", ipAddr.String())
		// Download files of key
		err := Download(ipAddr.String(), fmt.Sprint(Configuration.Service.filePort), filePath, juiceTempPath)
		if err != nil {
			Info.Println(err)
		}
		// Read and append data by new line
		content, err := ioutil.ReadFile(juiceTempPath)
		if err != nil {
			panic(err)
		}
		sorted = append(sorted, content...)
		// Remove file
		os.Remove(juiceTempPath)
	}
	// Return combined data
	return sorted
}

// (worker) executes juice call
func ExecuteJuice(exeName string, prefix string, key string, fruits []byte) error {
	Info.Println(exeName + " executed.")
	juiceCmd := exec.Command(exeName, juiceTempDir, prefix, key)
	juiceIn, err := juiceCmd.StdinPipe()
	if err != nil {
		return err
	}
	juiceCmd.Start()
	// Write map output data to pipe
	juiceIn.Write(fruits)
	juiceIn.Close()
	juiceCmd.Wait()
	Info.Println(exeName + " finished.")
	return nil
}
