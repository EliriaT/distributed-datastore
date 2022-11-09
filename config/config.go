package config

import (
	"encoding/json"
	"io"
	"log"
	"os"
)

var NodeConfig map[string]string

func Congif() {

	file, err := os.Open("./config/config.json")
	if err != nil {
		log.Fatal("Error opening config.json ", err)
	}
	defer file.Close()

	byteValue, _ := io.ReadAll(file)

	_ = json.Unmarshal(byteValue, &NodeConfig)

}
