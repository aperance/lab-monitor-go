package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/robertkrimen/otto"
)

type clientUpdate struct {
	ID          string                 `json:"id"`
	StateDiff   map[string]interface{} `json:"state"`
	HistoryDiff []interface{}          `json:"history"`
}

type device struct {
	ipAddress         string
	previousState     map[string]string
	currentState      map[string]string
	history           map[string][]interface{}
	status            string
	lastCommunication int64
	broadcast         chan *wsMessage
}

func (d *device) watch() {
	go func() {
		var sleepDuration time.Duration = 0
		var sequenceNumber string = "0"

		for {
			state, ok := d.fetchState(sequenceNumber)

			switch {
			case ok:
				if d.status != "CONNECTED" {
					log.Println(d.ipAddress + ": Connection established.")
				}
				d.currentState = state
				d.status = "CONNECTED"
				d.lastCommunication = time.Now().Unix()
				sequenceNumber = state[config.SequenceKey]
				sleepDuration = 0
			case d.status == "CONNECTED":
				log.Println(d.ipAddress + ": Connection lost. Retrying...")
				d.status = "RETRY"
				sequenceNumber = "0"
				sleepDuration = 0
			case d.status == "INACTIVE" || time.Now().Unix()-d.lastCommunication > 600000:
				log.Println(d.ipAddress + ": Device inactive. Retrying in 5 min.")
				d.status = "INACTIVE"
				sequenceNumber = "0"
				sleepDuration = 5 * time.Minute
			default:
				log.Println(d.ipAddress + ": Device disconnected. Retrying in 1 min.")
				d.status = "DISCONNECTED"
				sequenceNumber = "0"
				sleepDuration = 1 * time.Minute
			}

			if d.currentState != nil {
				d.currentState["status"] = d.status
				d.updateClients()
			}

			time.Sleep(sleepDuration)
			d.previousState = d.currentState
		}
	}()
}

func (d *device) fetchState(seq string) (map[string]string, bool) {
	url := "http://" + d.ipAddress + ":" + config.Port + "/" + config.Path + "?" + config.SequenceKey + "=" + seq
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return nil, false
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, false
	}

	vm := otto.New()
	vm.Set("data", string(body))
	value, err := vm.Run(`JSON.stringify(eval(data.replace("display(", "(")))`)
	if err != nil {
		log.Println(err)
		return nil, false
	}

	str, err := value.ToString()
	if err != nil {
		log.Println(err)
		return nil, false
	}

	state := make(map[string]string)
	err = json.Unmarshal([]byte(str), &state)
	if err != nil {
		log.Println(err)
		return nil, false
	}

	state["timestamp"] = fmt.Sprint(time.Now().Format("Mon Jan 2 3:04:05 PM"))

	return state, true
}

func (d *device) getStateDiff() map[string]interface{} {
	diff := make(map[string]interface{})

	if d.status == "CONNECTED" {
		for key, p := range d.previousState {
			c, ok := d.currentState[key]
			if ok == false {
				diff[key] = nil
			} else if c != p {
				diff[key] = c
			}
		}
		for key, c := range d.currentState {
			p, ok := d.previousState[key]
			if ok == false || c != p {
				diff[key] = c
			}
		}
	}

	return diff
}

func (d *device) updateClients() {
	s := d.getStateDiff()
	h := make([]interface{}, 0)
	d.broadcast <- &wsMessage{"DEVICE_DATA_UPDATE", clientUpdate{d.ipAddress, s, h}}
}
