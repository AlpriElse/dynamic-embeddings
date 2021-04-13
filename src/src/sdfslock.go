package main

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

type LockType int

const (
	SdfsRLock LockType = iota
	SdfsLock
)

type SdfsLockRequest struct {
	SessionId   int32
	RemoteFname string
	Type        LockType
}

type SdfsLockResponse struct {
	SessionId   int32
	RemoteFname string
	Type        LockType
}

type SdfsMutex struct {
	sync.Mutex
	Cond   *sync.Cond
	RCount int
}

const (
	lockTimeout = 60
)

var (
	sessionIdCt = new(int32)
)

// Create new SdfsMutex
func NewSdfsMutex() *SdfsMutex {
	mutex := SdfsMutex{}
	mutex.Cond = sync.NewCond(&mutex)
	return &mutex
}

// Get session Id
func NewSession() int32 {
	atomic.AddInt32(sessionIdCt, 1)
	sessionId := atomic.LoadInt32(sessionIdCt) % 100
	return sessionId
}

func (node *SdfsNode) RpcLock(sessionId int32, fname string, lType LockType) int32 {
	var lockRes SdfsLockResponse
	lockReq := SdfsLockRequest{SessionId: sessionId, RemoteFname: fname, Type: lType}
	err := client.Call("SdfsNode.AcquireLock", lockReq, &lockRes)
	if err != nil {
		panic(err)
	}
	return lockRes.SessionId
}

func (node *SdfsNode) RpcUnlock(sessionId int32, fname string, lType LockType) int32 {
	var lockRes SdfsLockResponse
	lockReq := SdfsLockRequest{SessionId: sessionId, RemoteFname: fname, Type: lType}
	err := client.Call("SdfsNode.ReleaseLock", lockReq, &lockRes)
	if err != nil {
		panic(err)
	}
	return lockRes.SessionId

}

// Acquire global lock for file
func (node *SdfsNode) AcquireLock(req SdfsLockRequest, reply *SdfsLockResponse) error {
	if node.isMaster == false && node.Master == nil {
		return errors.New("Error: Master not initialized")
	}

	sessId := NewSession()
	node.Master.sessMap[sessId] = make(chan bool, 1)

	node.Master.Lock(req.RemoteFname, req.Type)
	Info.Println("AcquireLock: Lock check wait ready")

	go func() {
		select {
		case <-node.Master.sessMap[sessId]:
			node.Master.Unlock(req.RemoteFname, req.Type)
			Info.Println("Lock released session confirm", sessId)
			return
		case <-time.After(lockTimeout * time.Second):
			Info.Println("Lock", sessId, "timed out after 30s")
			node.Master.Unlock(req.RemoteFname, req.Type)
			return
		}
	}()

	var resp SdfsLockResponse
	resp.SessionId = sessId
	resp.RemoteFname = req.RemoteFname
	*reply = resp
	return nil
}

// Release global lock for file
func (node *SdfsNode) ReleaseLock(req SdfsLockRequest, reply *SdfsLockResponse) error {
	if node.isMaster == false && node.Master == nil {
		return errors.New("Error: Master not initialized")
	}

	Info.Println("Releasing lock")
	node.Master.sessMap[req.SessionId] <- true
	Info.Println("Lock released")

	var resp SdfsLockResponse
	resp.RemoteFname = req.RemoteFname
	*reply = resp
	return nil
}

func (node *SdfsMaster) Lock(fname string, lockType LockType) {
	// If lock does not exist, create lock
	if _, ok := node.lockMap[fname]; !ok {
		node.lockMap[fname] = NewSdfsMutex()
		Info.Println("AcquireLock: Created new lock")
	}

	// Acquire RW lock for remoteFname mutex
	if lockType == SdfsRLock {
		// If read, try to lock
		node.lockMap[fname].Lock()
		node.lockMap[fname].RCount++
		node.lockMap[fname].Unlock()
	} else if lockType == SdfsLock {
		// If write (exclusive), can lock, respond
		node.lockMap[fname].Lock()
		for node.lockMap[fname].RCount > 0 {
			node.lockMap[fname].Cond.Wait()
		}
	}
}

func (node *SdfsMaster) Unlock(fname string, lockType LockType) {
	// If lock does not exist, error
	if _, ok := node.lockMap[fname]; !ok {
		panic("Error: Lock for file does not exist.")
	}

	// If read, try to unlock
	if lockType == SdfsRLock {
		node.lockMap[fname].Lock()
		node.lockMap[fname].RCount--
		if node.lockMap[fname].RCount == 0 {
			node.lockMap[fname].Cond.Signal()
		}
		node.lockMap[fname].Unlock()
	}

	// If write (exclusive), try to unlock
	if lockType == SdfsLock {
		node.lockMap[fname].Cond.Signal()
		node.lockMap[fname].Unlock()
	}
}
