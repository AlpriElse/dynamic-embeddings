package main

import (
	"hash/fnv"
	"sort"
)

// Hash or range partitioner
func partitioner(keys []string, numJuices int, isRange bool) map[int][]string {
	partitions := make(map[int][]string)
	// If range partition, sort list of keys
	if isRange {
		numParts := len(keys) / numJuices
		sort.Strings(keys)
		for i, k := range keys {
			partitions[(i/numParts)%numJuices] = append(partitions[(i/numParts)%numJuices], k)
		}
		return partitions
	}

	// If hash partition, enumerate list of keys and call getHashPartition
	for _, k := range keys {
		hash := getHashPartition(k, numJuices)
		partitions[hash] = append(partitions[hash], k)
	}
	return partitions
}

func getHashPartition(key string, numJuices int) int {
	p := int(getHash(key)) % numJuices
	return p
}

func getHash(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}
