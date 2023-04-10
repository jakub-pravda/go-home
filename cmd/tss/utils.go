package main

import (
	"encoding/json"
	"errors"
	"log"
)

// Parse input json string to SensorTrvSync struct
func parseSyncConfig(jsonStr string) (SensorTrvSync, error) {
	log.Printf("Parsing sync config: %s", jsonStr)
	var config SensorTrvSync
	err := json.Unmarshal([]byte(jsonStr), &config)
	if err != nil {
		log.Printf("Sync config parsing failed")
		return SensorTrvSync{}, err
	} else if config.SensorTopic == "" || config.TrvTopic == "" {
		log.Printf("Sync config parsing failed: sensor or TRV topic is empty")
		return SensorTrvSync{}, errors.New("sensor or TRV topic is empty")
	}
	return config, nil
}
