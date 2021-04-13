package main

import (
	"bufio"
	"fmt"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	// Configuration stores all info in config.json
	Configuration Config
	process       *Member
	sdfs          *SdfsNode
	client        *rpc.Client
)

func printOptions() {
	if process == nil {
		fmt.Println("Welcome! Don't be a loner and join the group by saying one of the following:\n" +
			"-	join introducer\n" +
			"-	join")
	} else {
		fmt.Println("\nGroup Interaction Options:")
		fmt.Println("-	leave | kill | stop | switch [gossip/alltoall]")

		fmt.Println("\nQuery Group Information:")
		fmt.Println("-	status | whoami | print logs {-n} | grep {all} [query]")

		fmt.Println("\nResource Monitoring:")
		fmt.Println("-	metrics | sim failtest")

		fmt.Println("\nSDFS Commands:")
		fmt.Println("-	put [local file] [sdfsfile]")
		fmt.Println("-	get [sdfs file] [local file]")
		fmt.Println("-	delete [sdfs file]")
		fmt.Println("-	ls")
		fmt.Println("-	store")

		fmt.Println("\nMapleJuice Commands:")
		fmt.Println("-  maple [local file] [num of tasks] [sdfs file] [sdfs dir]")
		fmt.Println("-  juice [local file] [num of tasks] [sdfs file] [sdfs dir]")
	}
}

func StartCli() {
	printOptions()
	consoleReader := bufio.NewReader(os.Stdin)
	for {
		// wait for input to query operations on node
		fmt.Print("> ")
		input, _ := consoleReader.ReadString('\n')
		input = strings.TrimSuffix(input, "\n")
		inputFields := strings.Fields(input) // Split string into os.Args like array

		if len(inputFields) == 0 {
			fmt.Println("invalid command")
			continue
		}

		switch inputFields[0] {
		case "join":
			if process != nil && sdfs != nil {
				Warn.Println("You have already joined!")
				continue
			}

			if len(inputFields) == 2 && inputFields[1] == "introducer" {
				process = InitMembership(true)
				sdfs = InitSdfs(process, true)
				// initialize file transfer server
				go InitFileTransferServer(fmt.Sprint(Configuration.Service.filePort))

			} else {
				// Temporarily, the memberID is 0, will be set to correct value when introducer adds it to group
				process = InitMembership(false)
				sdfs = InitSdfs(process, false)
				go InitFileTransferServer(fmt.Sprint(Configuration.Service.filePort))
			}

			// start gossip
			if process == nil {
				Warn.Println("You need to join group before you start.")
				continue
			}

			go process.Tick()

		case "leave":
			if process == nil {
				Warn.Println("You need to join in order to leave!")
				continue
			}
			process.leave()
			listener.Close()
			Info.Println("Node has left the group.")
			process = nil

		case "kill":
			Warn.Println("Killing process. Bye bye.")
			os.Exit(1)

		case "status":
			if process == nil {
				Warn.Println("You need to join in order to get status!")
				continue
			}

			process.PrintMembershipList(os.Stdout)

		case "print":
			if len(inputFields) >= 2 && inputFields[1] == "logs" {
				if len(inputFields) == 4 && inputFields[2] == "-n" {
					num, err := strconv.Atoi(inputFields[3])
					if err != nil {
						fmt.Println("Please provide a valid number of lines")
						continue
					}

					printLogs(num)

				} else {
					printLogs(0)
				}
			}

		case "grep":
			if process == nil {
				Warn.Println("You need to join in order to get status!")
				continue
			}

			if len(inputFields) >= 2 {
				if len(inputFields) >= 3 && inputFields[1] == "all" {
					process.Grep(strings.Join(inputFields[2:], " "), false)
					// sleep for a tiny bit of time so that you get all results before restarting loop
					time.Sleep(250 * time.Millisecond)
				} else {
					process.Grep(strings.Join(inputFields[1:], " "), true)
				}
			}

		case "stop":
			process.StopTick()

		case "switch":
			if len(inputFields) >= 2 && inputFields[1] == "gossip" {
				SetHeartbeating(true)
				process.SendAll(SwitchMsg, []byte{1})
			} else if len(inputFields) >= 2 && inputFields[1] == "alltoall" {
				SetHeartbeating(false)
				process.SendAll(SwitchMsg, []byte{0})
			}

		// Monitoring
		case "metrics":
			if memMetrics == nil {
				InitMonitor()
			}
			memMetrics.PrintMonitor()

		case "whoami":
			if process == nil {
				Warn.Println("You need to join group before you are assigned an ID.")
				continue
			}

			fmt.Println("You are member " + fmt.Sprint(process.memberID))

		case "sim":
			if len(inputFields) >= 2 && inputFields[1] == "failtest" {
				process.SendAll(TestMsg, []byte{})
			}

		// SDFS
		case "put":
			if len(inputFields) >= 3 && process != nil {
				if client == nil || sdfs == nil {
					Warn.Println("Client not initialized.")
					continue
				}
				sessionId := sdfs.RpcLock(int32(sdfs.Member.memberID), inputFields[2], SdfsLock)
				sdfs.RpcPut(inputFields[1], inputFields[2])
				sessionId = sdfs.RpcUnlock(sessionId, inputFields[2], SdfsLock)

				fmt.Println("Finished put.")
			}

		case "get":
			if len(inputFields) >= 3 {
				if client == nil || sdfs == nil {
					Warn.Println("Client not initialized.")
					continue
				}
				sessionId := sdfs.RpcLock(int32(sdfs.Member.memberID), inputFields[1], SdfsRLock)
				sdfs.RpcGet(inputFields[1], inputFields[2])
				sessionId = sdfs.RpcUnlock(sessionId, inputFields[1], SdfsRLock)
				fmt.Println("Finished get.")
			}

		case "delete":
			if len(inputFields) >= 2 {
				if client == nil || sdfs == nil {
					Warn.Println("Client not initialized.")
					continue
				}
				sessionId := sdfs.RpcLock(int32(sdfs.Member.memberID), inputFields[1], SdfsLock)
				sdfs.RpcDelete(inputFields[1])
				sessionId = sdfs.RpcUnlock(sessionId, inputFields[1], SdfsLock)
			}

		case "ls":
			if len(inputFields) >= 2 {
				if client == nil {
					Warn.Println("Client not initialized.")
					continue
				}
				sdfs.RpcListIPs(inputFields[1])
			}

		case "store":
			if sdfs == nil {
				fmt.Println("SDFS not initialized.")
				continue
			}
			sdfs.Store()

		case "upload":
			if len(inputFields) == 4 {
				fileContents, err := GetFileContents(inputFields[2])
				if err != nil {
					fmt.Println("File does not exist.")
					continue
				}
				Upload(fmt.Sprint(inputFields[1]),
					fmt.Sprint(Configuration.Service.filePort),
					inputFields[2],
					inputFields[3],
					fileContents)
			}

		case "download":
			if len(inputFields) == 4 {
				err := Download(fmt.Sprint(inputFields[1]),
					fmt.Sprint(Configuration.Service.filePort),
					inputFields[2],
					inputFields[3])
				if err != nil {
					fmt.Println(err)
				}
			}

		case "master":
			if sdfs != nil {
				fmt.Println(sdfs.MasterId)
			}

		case "maple":
			if process != nil && sdfs != nil && len(inputFields) >= 5 {
				num, err := strconv.Atoi(inputFields[2])
				if err != nil {
					fmt.Println("Error: num_maples must be a number.")
					continue
				}

				err = sdfs.QueueTask(MapleJuiceQueueRequest{
					RequestingId:       process.memberID,
					IsMaple:            true,
					FileList:           GetFileNames(inputFields[4]),
					ExeName:            inputFields[1],
					NumTasks:           num,
					IntermediatePrefix: inputFields[3]})
				if err != nil {
					fmt.Println("Error in queueing task: ", err)
				}
			}

		case "juice":
			if process != nil && len(inputFields) >= 6 {
				tasks, err := strconv.Atoi(inputFields[2])
				if err != nil {
					fmt.Println("Error: num_juice must be a number.")
				}

				del := false
				if inputFields[5] == "1" {
					del = true
				}

				var dirName []string
				dirName = append(dirName, inputFields[4])

				sdfs.QueueTask(MapleJuiceQueueRequest{
					RequestingId:       process.memberID,
					IsMaple:            false,
					ExeName:            inputFields[1],
					NumTasks:           tasks,
					IntermediatePrefix: inputFields[3],
					FileList:           dirName,
					DeleteInput:        del})
			}
		case "inspectMaster":
			if process != nil && process.memberID == sdfs.MasterId {
				fmt.Println("filemap", sdfs.Master.fileMap)
				fmt.Println("keylocations", sdfs.Master.keyLocations)
				fmt.Println("prefixKeyMap", sdfs.Master.prefixKeyMap)
				fmt.Println("sdfsFNameMap", sdfs.Master.sdfsFNameMap)
				fmt.Println("numBlocks", sdfs.Master.numBlocks)
			}

		case "help":
			printOptions()
		default:
			fmt.Println("invalid command")
			printOptions()
		}
	}
}

func main() {
	// Set up loggers and configs
	InitLog()
	InitMonitor()
	Configuration = ReadConfig()
	Configuration.Print()
	InitDirectories()
	StartCli()

}
