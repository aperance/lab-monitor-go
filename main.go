package main

import (
	"fmt"
	"net/http"
	"strings"
)

type accumulatedRecords struct {
	State   map[string]map[string]string        `json:"state"`
	History map[string]map[string][]interface{} `json:"histroy"`
}

var devicePool map[string]*device

func getAccumulatedRecords() accumulatedRecords {
	stateMap := make(map[string]map[string]string)
	historyMap := make(map[string]map[string][]interface{})

	for ipAddress, device := range devicePool {
		if state := device.currentState; state != nil {
			stateMap[ipAddress] = state
		}
		if history := device.history; history != nil {
			historyMap[ipAddress] = history
		}
	}

	return accumulatedRecords{stateMap, historyMap}
}

func start(p *wsPool) {
	devicePool = make(map[string]*device)

	for _, r := range config.AddressRanges {
		for i := r.Start; i <= r.End; i++ {
			ipAddress := strings.TrimSuffix(r.Subnet, "0") + fmt.Sprint(i)
			d := device{
				ipAddress: ipAddress,
				broadcast: p.broadcast,
			}
			devicePool[ipAddress] = &d
			d.watch()
		}
	}
}

func main() {

	p := newPool()
	go p.start()

	start(p)

	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		serveWs(p, w, r)
	})
	http.ListenAndServe(":8080", nil)
}
