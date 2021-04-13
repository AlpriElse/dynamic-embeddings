package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// Config struct
type Config struct {
	Service  Service  `json:"service"`
	Settings Settings `json:"settings"`
}

// Service struct
type Service struct {
	detectorType string  `json:"failure_detector"`
	introducerIP string  `json:"introducer_ip"`
	port         float64 `json:"port"`
	masterIP     string  `json:"initial_master_ip"`
	masterPort   float64 `json:"master_port"`
	filePort     float64 `json:"file_port"`
}

// Settings struct
type Settings struct {
	gossipInterval       float64 `json:"gossip_interval"`
	allInterval          float64 `json:"all_interval"`
	failTimeout          float64 `json:"fail_timeout"`
	cleanupTimeout       float64 `json:"cleanup_timeout"`
	numProcessesToGossip float64 `json:"num_processes_to_gossip"`
	replicationFactor    float64 `json:"replication_factor"`
	blockSize            float64 `json:"block_size_mb"`
}

// ReadConfig function to read the configuration JSON
func ReadConfig() Config {
	jsonFile, err := os.Open("../config.json")

	if err != nil {
		fmt.Println(err)
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var result map[string]interface{}
	json.Unmarshal([]byte(byteValue), &result)

	// Create service struct
	serviceJSON := result["service"].(map[string]interface{})
	detectType := serviceJSON["failure_detector"].(string)
	addr := serviceJSON["introducer_ip"].(string)
	masterPort := serviceJSON["master_port"].(float64)
	masterIP := serviceJSON["initial_master_ip"].(string)
	port := serviceJSON["port"].(float64)
	filePort := serviceJSON["file_port"].(float64)

	service := Service{
		detectorType: detectType,
		introducerIP: addr,
		port:         port,
		masterIP:     masterIP,
		masterPort:   masterPort,
		filePort:     filePort,
	}

	// Create settings struct
	settingsJSON := result["settings"].(map[string]interface{})
	gInterval := settingsJSON["gossip_interval"].(float64)
	aInterval := settingsJSON["all_interval"].(float64)
	fTime := settingsJSON["fail_timeout"].(float64)
	cTime := settingsJSON["cleanup_timeout"].(float64)
	numProcessesToGossip := settingsJSON["num_processes_to_gossip"].(float64)
	rFactor := settingsJSON["replication_factor"].(float64)
	blockSize := settingsJSON["block_size_mb"].(float64)

	settings := Settings{
		gossipInterval:       gInterval,
		allInterval:          aInterval,
		failTimeout:          fTime,
		cleanupTimeout:       cTime,
		numProcessesToGossip: numProcessesToGossip,
		replicationFactor:    rFactor,
		blockSize:            blockSize,
	}

	config := Config{
		Service:  service,
		Settings: settings,
	}
	return config
}

func (c *Config) Print() {
	Info.Println("Detector: " + c.Service.detectorType)
	Info.Println("Introducer: " + c.Service.introducerIP + " on port " + fmt.Sprint(c.Service.port))
	Info.Println("Gossip interval: " + fmt.Sprint(c.Settings.gossipInterval))
	Info.Println("All-to-All interval: " + fmt.Sprint(c.Settings.allInterval))
	Info.Println("Failure timeout: " + fmt.Sprint(c.Settings.failTimeout))
	Info.Println("Cleanup timeout: " + fmt.Sprint(c.Settings.cleanupTimeout))
}

func healthEnumToString(healthEnum uint8) string {
	switch healthEnum {
	case 0:
		return "Alive"
	case 1:
		return "Failed"
	case 2:
		return "Left"
	}
	return "Invalid"
}
