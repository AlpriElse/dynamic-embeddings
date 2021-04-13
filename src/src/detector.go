package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"math/rand"
	"net"
	"text/tabwriter"
	"time"
)

// Member struct to hold member info
type Member struct {
	memberID       uint8
	isIntroducer   bool
	membershipList map[uint8]membershipListEntry // {uint8 (member_id): Member}
}

// holds one entry of the membership list
type membershipListEntry struct {
	MemberID       uint8
	IPaddr         net.IP
	HeartbeatCount uint64
	Timestamp      time.Time
	Health         uint8 // -> Health enum
}

// Health enum
const (
	Alive = iota
	Failed
	Left
)

// Ticker variables
var (
	joinAck      = make(chan bool)
	disableHeart = make(chan bool)
	failCh       = make(chan uint8, 10)
	ticker       *time.Ticker
	enabledHeart = false
	isGossip     = true
	listener     *net.UDPConn
	maxID        uint8 = 0
)

// Initialize membership protocol
func InitMembership(setIntroducer bool) *Member {
	mem := NewMember(setIntroducer)
	if Configuration.Service.detectorType == "alltoall" {
		isGossip = false
	}

	// initialize heartbeat listener
	go mem.Listen(fmt.Sprint(Configuration.Service.port))
	if setIntroducer {
		mem.membershipList[0] = NewMembershipListEntry(0, net.ParseIP(Configuration.Service.introducerIP))
		Info.Println("You are now the introducer.")
		return mem
	}
	time.Sleep(100 * time.Millisecond) // Sleep a tiny bit so listener can start
	mem.joinRequest()
	// Wait for response
	select {
	case _ = <-joinAck:
		fmt.Println("Node has joined the group.")
	case <-time.After(2 * time.Second):
		fmt.Println("Timeout join. Please retry again.")
		listener.Close()
		mem = nil
	}
	return mem
}

// Member constructor
func NewMember(introducer bool) *Member {
	mem := &Member{
		0,
		introducer,
		make(map[uint8]membershipListEntry),
	}
	return mem
}

// membershipListEntry constructor
func NewMembershipListEntry(memberID uint8, address net.IP) membershipListEntry {
	mlEntry := membershipListEntry{
		memberID,
		address,
		0,
		time.Now(),
		Alive,
	}
	return mlEntry
}

// PrintMembershipList pretty-prints all values inside the membership list
func (mem *Member) PrintMembershipList(output io.Writer) {
	fmt.Println("Current time: ", time.Now())
	writer := tabwriter.NewWriter(output, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "MemberID\tIP\tHeartbeats\tTimestamp\tHealth")
	fmt.Fprintln(writer, "-------\t----------\t----\t-----------\t------")
	for _, v := range mem.membershipList {
		fmt.Fprintf(writer, "%v\t%v\t%v\t%v\t%v\n", v.MemberID, v.IPaddr, v.HeartbeatCount, v.Timestamp.Format("15:04:05.000"), healthEnumToString(v.Health))
	}
	writer.Flush()
}

// Verifies whether a node has been updated in T_Failed seconds.
func (mem *Member) FailMember(memberId uint8, oldTime time.Time) {
	if memberId == mem.memberID {
		return
	}

	time.Sleep(time.Duration(Configuration.Settings.failTimeout) * time.Second)

	if currEntry, ok := mem.membershipList[memberId]; ok {
		difference := time.Now().Sub(currEntry.Timestamp)
		threshold := time.Duration(Configuration.Settings.failTimeout) * time.Second
		if difference >= threshold {
			if currEntry.Health == Alive {
				memMetrics.Increment(numFailures, 1)
				mem.membershipList[memberId] = membershipListEntry{
					currEntry.MemberID,
					currEntry.IPaddr,
					currEntry.HeartbeatCount,
					currEntry.Timestamp,
					Failed,
				}
				failCh <- memberId
				Info.Println("Marked member failed: ", memberId,
					"\nFail time: ", time.Now(),
					"\nOld time: ", oldTime)
			}

			go mem.CleanupMember(memberId, oldTime)
		}
	}

}

// Verifies whether a node has been updated in T_Cleanup seconds.
func (mem *Member) CleanupMember(memberId uint8, oldTime time.Time) {
	if memberId == mem.memberID {
		return
	}

	time.Sleep(time.Duration(Configuration.Settings.cleanupTimeout) * time.Second)

	if currEntry, ok := mem.membershipList[memberId]; ok {
		difference := time.Now().Sub(currEntry.Timestamp)
		threshold := time.Duration(Configuration.Settings.cleanupTimeout) * time.Second
		if difference >= threshold {
			delete(mem.membershipList, memberId)
			Info.Println("Cleaned up member: ", memberId)
		}
	}
}

// Invoked when hearbeat is recieved
func (mem *Member) HeartbeatHandler(membershipListBytes []byte) {
	// grab membership list in order to merge with your own
	// decode the buffer to the membership list, similar to joinResponse()
	b := bytes.NewBuffer(membershipListBytes)
	d := gob.NewDecoder(b)
	rcvdMemList := make(map[uint8]membershipListEntry)

	err := d.Decode(&rcvdMemList)
	if err != nil {
		fmt.Println(err)
		return
	}

	for id, rcvdEntry := range rcvdMemList {
		// If somebody thinks you are a failure then quit and rejoin :(
		if id == mem.memberID {
			if rcvdEntry.Health == Failed && !rcvdEntry.IPaddr.Equal(net.ParseIP(Configuration.Service.introducerIP)) {
				mem.joinRequest()
			} else if rcvdEntry.Health == Failed && rcvdEntry.IPaddr.Equal(net.ParseIP(Configuration.Service.introducerIP)) {
				// if introducer is falsely detected as failed then don't rejoin, just give new ID
				newID := getMaxID()
				newEntry := mem.membershipList[mem.memberID]
				newEntry.MemberID = newID
				delete(mem.membershipList, mem.memberID)
				mem.membershipList[newID] = newEntry
				mem.memberID = newID
			}
			continue
		}

		doUpdate := true

		// check that they have the same id in their membership list
		if currEntry, ok := mem.membershipList[id]; ok {
			// No changes to timestamp/heartbeat count if count has not been updated or entry left
			if rcvdEntry.HeartbeatCount <= currEntry.HeartbeatCount && rcvdEntry.Health != Left {
				doUpdate = false
			}
			// don't update a failed entry
			if rcvdEntry.Health == Failed && mem.membershipList[id].Health == Failed {
				doUpdate = false
			}
		} else {
			// don't add new failed entries
			if rcvdEntry.Health == Failed {
				doUpdate = false
			}
		}

		// Only set if update is neccessary whatsoever or if new entry to add
		if doUpdate {
			mem.membershipList[id] = membershipListEntry{
				rcvdEntry.MemberID,
				rcvdEntry.IPaddr,
				rcvdEntry.HeartbeatCount,
				time.Now(),
				rcvdEntry.Health,
			}
		}
		oldTime := time.Now()

		// Cmp most recently updated entry timestamp
		go mem.FailMember(id, oldTime)
	}
}

func setTicker() {
	ticker = time.NewTicker(time.Duration(Configuration.Settings.gossipInterval) * 1000 * time.Millisecond)
}

// Timer to schedule heartbeats
func (mem *Member) Tick() {
	if ticker == nil {
		setTicker()
	}

	if enabledHeart {
		Warn.Println("Heartbeating has already started.")
		return
	}

	memMetrics.StartMonitor()
	enabledHeart = true
	for {
		// Listen channel to disable heartbeating
		select {
		case <-disableHeart:
			enabledHeart = false
			Warn.Println("Stopped heartbeating.")
			return
		case _ = <-ticker.C:
			// Increment heartbeat counter of self
			entry := mem.membershipList[mem.memberID]
			entry.HeartbeatCount += 1
			entry.Timestamp = time.Now()
			mem.membershipList[mem.memberID] = entry
			// Gossip or AllToAll
			if isGossip {
				for i := 0.0; i < Configuration.Settings.numProcessesToGossip; i++ {
					mem.Gossip()
				}
			} else {
				mem.AllToAll()
			}
		}
	}
}

// Stop ticking if enabled
func (mem *Member) StopTick() {
	if !enabledHeart {
		Warn.Println("No process running to stop.")
		return
	}

	disableHeart <- true
}

// Switch heartbeating modes (All to All or Gossip)
func SetHeartbeating(flag bool) {
	if ticker == nil {
		setTicker()
	}

	if isGossip == flag {
		if isGossip {
			Warn.Println("You are already running Gossip")
		} else {
			Warn.Println("You are already running All to All")
		}
		return
	}

	isGossip = flag
	interval := time.Millisecond
	if isGossip {
		Info.Println("Running Gossip at T =", Configuration.Settings.gossipInterval)
		interval = time.Duration(Configuration.Settings.gossipInterval) * 1000 * interval
	} else {
		Info.Println("Running All-to-All at T =", Configuration.Settings.allInterval)
		interval = time.Duration(Configuration.Settings.allInterval) * 1000 * interval
	}

	ticker.Reset(interval)
}

// Send Gossip to random member in group
func (mem *Member) Gossip() {
	// Select random member
	addr := mem.PickRandMemberIP()
	if addr == nil {
		Info.Println("No other member to gossip to!")
		return
	}

	//Info.Println("Gossiping to " + addr.String())

	// Encode the membership list to send it
	b := new(bytes.Buffer)
	e := gob.NewEncoder(b)
	err := e.Encode(mem.membershipList)
	if err != nil {
		panic(err)
	}

	Send(addr.String()+":"+fmt.Sprint(Configuration.Service.port), HeartbeatMsg, b.Bytes())
}

// AllToAll heartbeating - sends membership list to all other
func (mem *Member) AllToAll() {
	// Encode the membership list to send it
	b := new(bytes.Buffer)
	e := gob.NewEncoder(b)
	err := e.Encode(mem.membershipList)
	if err != nil {
		panic(err)
	}
	//Info.Println("Sending All-to-All.")
	// Send heartbeatmsg and membership list to all members
	mem.SendAll(HeartbeatMsg, b.Bytes())
}

// Broadcast a message to all members in membershiplist
func (mem *Member) SendAll(msgType MessageType, msg []byte) {
	for _, v := range mem.membershipList {
		Send(v.IPaddr.String()+":"+fmt.Sprint(Configuration.Service.port), msgType, msg)
	}
}

// request introducer to join
func (mem *Member) joinRequest() {
	Send(Configuration.Service.introducerIP+":"+fmt.Sprint(Configuration.Service.port), JoinMsg, nil)
}

// receive membership list from introducer and setup
func (mem *Member) joinResponse(membershipListBytes []byte) {
	// First byte received corresponds to assigned memberID
	mem.memberID = uint8(membershipListBytes[0])

	// clear existing membership list
	mem.membershipList = make(map[uint8]membershipListEntry)

	// Decode the rest of the buffer to the membership list
	b := bytes.NewBuffer(membershipListBytes[1:])
	d := gob.NewDecoder(b)
	err := d.Decode(&mem.membershipList)
	if err != nil {
		panic(err)
	}
	joinAck <- true

	Info.Println(mem.membershipList)
}

// modify membership list entry
func (mem *Member) leave() {
	newEntry := mem.membershipList[mem.memberID]
	newEntry.HeartbeatCount++
	newEntry.Health = Left
	newEntry.Timestamp = time.Now()
	mem.membershipList[mem.memberID] = newEntry
	// Gossip leave status and stop
	mem.Gossip()
	mem.StopTick()
	Info.Println("Leaving group: ", mem.memberID)
}

// for introducer to accept a new member
func (mem *Member) acceptMember(address net.IP) {
	// assign new ID
	newMemberID := getMaxID()
	mem.membershipList[newMemberID] = NewMembershipListEntry(newMemberID, address)

	// Encode the membership list to send it
	b := new(bytes.Buffer)
	e := gob.NewEncoder(b)
	err := e.Encode(mem.membershipList)
	if err != nil {
		panic(err)
	}

	// Send the memberID by appending it to start of buffer, and the membershiplist
	Send(address.String()+":"+fmt.Sprint(Configuration.Service.port), AcceptMsg, append([]byte{newMemberID}, b.Bytes()...))
}

func (mem *Member) GetNumAlive() int {
	numAlive := 0
	for _, entry := range mem.membershipList {
		if entry.Health == Alive {
			numAlive += 1
		}
	}

	return numAlive
}

// getMaxID to get the maximum of all memberIDs
func getMaxID() uint8 {
	maxID++
	return maxID
}

// Find random IP in membership list
func (mem *Member) PickRandMemberIP() net.IP {
	if len(mem.membershipList) == 1 {
		// you are the only process in the list
		return nil
	}

	// loop until you find a member that isn't your own
	for {
		i := 0
		randVal := rand.Intn(len(mem.membershipList))
		var randEntry membershipListEntry
		for _, v := range mem.membershipList {
			if i == randVal {
				randEntry = v
			}

			i++
		}

		if randEntry.MemberID != mem.memberID {
			return randEntry.IPaddr
		}
	}
	return nil
}
