package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"strings"
)

type MessageType uint8

const (
	JoinMsg = iota
	HeartbeatMsg
	TextMsg
	AcceptMsg
	GrepReq
	GrepResp
	SwitchMsg
	TestMsg

	// Leader Election
	ElectionMsg
	CoordinatorMsg
	OkMsg

	// Recovery
	RecoverMasterMsg
)

// Debugging consts
var (
	dropMessage = false
	dropRate    = 0
)

// Send text message over UDP given address and string
func SendMessage(address string, msg string) {
	Send(address, TextMsg, []byte(msg))
}

// Broadcast message over UDP given addresses, messagetype, msg
func SendBroadcast(addresses []string, msgType MessageType, msg []byte) {
	for _, addr := range addresses {
		Send(addr, msgType, msg)
	}
}

// Sends message over UDP given address, messagetype, msg
func Send(address string, msgType MessageType, msg []byte) {
	// Debug purposes: simulate message drop
	if dropMessage && rand.Intn(100) < dropRate {
		memMetrics.Increment(messageDrop, 1)
		return
	}
	memMetrics.Increment(messageSent, 1)
	memMetrics.Increment(bytesSent, int64(len(msg)))

	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		panic(err)
	}

	// Get UDP "connection"
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		panic(err)
	}

	// Encoded into a byte buffer of the structure:
	// MessageType uint8
	// Message byte[]
	// [0] - MessageType, [1, ...] - message
	buffer := append([]byte{byte(msgType)}, msg...)

	_, err = conn.Write(buffer)
	if err != nil {
		panic(err)
	}
}

// Listen function to keep listening for messages
func (mem *Member) Listen(port string) {
	// UDP buffer 1024 bytes for now
	buffer := make([]byte, 1024)
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		panic(err)
	}

	listener, err = net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}

	// listener loop
	for {
		n, senderAddr, err := listener.ReadFromUDP(buffer)
		if err != nil {
			return
		}

		msgType := buffer[0]
		memMetrics.Increment(bytesReceived, int64(n))

		switch msgType {
		case TextMsg:
			fmt.Println(string(buffer[1:n]))
		case JoinMsg: // only introducer can accept join messages
			if mem.isIntroducer == true {
				Info.Println(senderAddr.String() + " requests to join.")
				mem.acceptMember(senderAddr.IP)
			}
		case HeartbeatMsg: // handles receipt of heartbeat
			mem.HeartbeatHandler(buffer[1:n])
			//Info.Println("Recieved heartbeat from ", senderAddr.String())
		case AcceptMsg: // handles receipt of membership list from introducer
			Info.Println("Introducer has accepted join request.")
			mem.joinResponse(buffer[1:n])
		case GrepReq: // handles grep request
			ipAddr := senderAddr.String()[:strings.IndexByte(senderAddr.String(), ':')]
			mem.HandleGrepRequest(ipAddr, buffer[1:n])
		case GrepResp: // handles grep response when one is received
			mem.HandleGrepResponse(buffer[1:n])
		case SwitchMsg:
			if buffer[1] == 1 {
				SetHeartbeating(true)
			} else {
				SetHeartbeating(false)
			}
		case TestMsg:
			memMetrics.PerfTest()
		default:
			Warn.Println("Member: Invalid message type")
		}
	}
}

func (node *SdfsNode) ListenSdfs(port string) {
	// UDP buffer 1024 bytes for now
	buffer := make([]byte, 1024)
	addr, err := net.ResolveUDPAddr("udp", ":"+port)
	if err != nil {
		panic(err)
	}

	sdfsListener, err = net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}

	// Listen for failures from membership list
	go node.MemberListen()

	// listener loop
	for {
		n, senderAddr, err := sdfsListener.ReadFromUDP(buffer)
		if err != nil {
			return
		}

		msgType := buffer[0]

		switch msgType {
		case ElectionMsg:
			node.handleElection(senderAddr.IP, buffer[1])
		case CoordinatorMsg:
			node.handleCoordinator(buffer[1])
		case OkMsg:
			node.handleOk()
		case RecoverMasterMsg:
			node.handleRecoverMaster(senderAddr.IP, buffer[1:n])
		default:
			Warn.Println("Sdfs: Invalid message type")
		}
	}
}

func (node *SdfsNode) closeRPCClient() {
	if client == nil {
		return
	}

	err := client.Close()
	if err != nil {
		panic(err)
	}
}

func (node *SdfsNode) startRPCClient(serverIP string, port string) {
	var err error
	client, err = rpc.DialHTTP("tcp", serverIP+":"+port)
	if err != nil {
		fmt.Println("Connection error: ", err)
	}
}

func (node *SdfsNode) startRPCServer(port string) {
	err := rpc.Register(node)
	if err != nil {
		fmt.Println("Format isn't correct. ", err)
	}
	rpc.HandleHTTP()
	rpcListener, e := net.Listen("tcp", ":"+port)
	if e != nil {
		fmt.Println("error in starting listener")
	}

	fmt.Printf("Serving RPC server on port %f\n", port)
	// Start accepting incoming HTTP connections
	go http.Serve(rpcListener, nil)
}
