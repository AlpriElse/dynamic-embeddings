package main

import (
	"fmt"
	"time"
)

type MemberMetrics struct {
	startTime time.Time
	metrics   map[uint8]int64
}

const (
	messageDrop uint8 = iota
	messageSent
	numFailures
	bytesReceived
	bytesSent
)

var (
	memMetrics *MemberMetrics
)

// Increment metric types
func (met *MemberMetrics) Increment(metricType uint8, value int64) int64 {
	if _, ok := met.metrics[metricType]; ok {
		met.metrics[metricType] += value
		return met.metrics[metricType]
	}
	met.metrics[metricType] = value
	return value
}

// Initialize global monitor
func InitMonitor() {
	memMetrics = &MemberMetrics{
		time.Now(),
		make(map[uint8]int64),
	}
	memMetrics.StartMonitor()
}

// Reinitialize start time of monitor
func (met *MemberMetrics) StartMonitor() {
	met.startTime = time.Now()
	met.metrics = make(map[uint8]int64)
	met.metrics[messageDrop] = 0
	met.metrics[messageSent] = 0
	met.metrics[numFailures] = 0
	met.metrics[bytesReceived] = 0
	met.metrics[bytesSent] = 0
}

// Print monitor values
func (met *MemberMetrics) PrintMonitor() {
	// messages dropped / messages sent
	// # failures
	// time.Now() - met.startTime
	// failures / node / second
	// total failures / node / second
	timeElapsed := time.Now().Sub(met.startTime)
	messageLoss := float64(met.metrics[messageDrop]) / float64(met.metrics[messageSent])
	failRate := float64(met.metrics[numFailures]) / float64(int64(timeElapsed/time.Second))
	bandwidth := float64(met.metrics[bytesReceived]) / float64(int64(timeElapsed/time.Second))
	bandwidthOut := float64(met.metrics[bytesSent]) / float64(int64(timeElapsed/time.Second))

	fmt.Println("Elapsed time (s): ", timeElapsed, "s")
	fmt.Println("Failures detected: ", met.metrics[numFailures])
	fmt.Println("Messages sent: ", met.metrics[messageSent])
	fmt.Println("Messages dropped: ", met.metrics[messageDrop])
	fmt.Println("Message loss rate: ", messageLoss)
	fmt.Println("Failure rate (f/s): ", failRate)
	fmt.Println("Bandwidth usage (bytes/s received): ", bandwidth)
	fmt.Println("Bandwidth usage (bytes/s sent): ", bandwidthOut)
}

func (met *MemberMetrics) PerfTest() {
	// for wait loop with sleep
	// gradually increase message loss rate by ~4
	// PrintMonitor after 3 minutes
	fmt.Println("Starting automated failure detection test...")
	dropMessage = true
	go func() {
		for i := 0; i < 5; i++ {
			met.StartMonitor()
			if i != 0 {
				dropRate += 5
			}
			fmt.Println("Setting simulated drop rate to: ", dropRate)
			time.Sleep(180 * time.Second)
			met.PrintMonitor()
		}
		fmt.Println("Finished failure detection test!")
		dropMessage = false
	}()
}
