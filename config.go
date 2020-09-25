package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

type configuration struct {
	AddressRanges []struct {
		Subnet string `json:"subnet"`
		Start  int    `json:"start"`
		End    int    `json:"end"`
	} `json:"addressRanges"`
	Port        string `json:"port"`
	Path        string `json:"path"`
	SequenceKey string `json:"sequenceKey"`
	MaxRetries  int    `json:"maxRetries"`
}

var config configuration

func init() {
	jsonFile, err := os.Open("config.json")
	if err != nil {
		log.Println(err)
	}
	defer jsonFile.Close()
	bytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Println(err)
	}
	if err = json.Unmarshal(bytes, &config); err != nil {
		log.Println(err)
	}

	log.Printf("%v", config)
}
