package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func main() {
	config := &Config{}

	data, err := os.ReadFile("config/config.json")
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	err = json.Unmarshal(data, config)
	if err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}

	fmt.Printf("Loaded config: %+v\n", config)
}
