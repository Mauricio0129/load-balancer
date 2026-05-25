package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func main() {
	config_data := &Config{}

	data, err := os.ReadFile("config/config.json")
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	err = json.Unmarshal(data, config_data)
	if err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}

	fmt.Printf("Loaded config successfully")

	startLoadBalancer(config_data)
}
