package cluster

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/EliriaT/distributed-datastore/store"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func GetObject(w http.ResponseWriter, r *http.Request) {
	var value []byte
	vars := mux.Vars(r)
	key, ok := vars["key"]
	if !ok {
		http.Error(w, errors.New("wrong query name").Error(), http.StatusBadRequest)
		return
	}

	peersCommand := Command{
		Key:         key,
		Value:       "",
		CommandType: GET,
	}

	byteMsg, err := json.Marshal(peersCommand)
	if err != nil {
		log.Fatal(err)
	}

	originalNode, replicaNode := GetShardAndReplica(key)
	var tcpAddres string

	if originalNode.Id == GetNode().Id || replicaNode.Id == GetNode().Id {
		value, err = store.NodeDataStore.GetValue(key)
		if originalNode.Id == GetNode().Id {
			tcpAddres = replicaNode.Name + replicaNode.TcpPort
		} else {
			tcpAddres = originalNode.Name + originalNode.TcpPort
		}
		if err != nil {
			receivedCommand := SendTCPRequest(byteMsg, tcpAddres)
			if receivedCommand.Value == "" {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			value = []byte(receivedCommand.Value)
		}
	} else {
		receivedCommand := SendTCPRequest(byteMsg, originalNode.Name+originalNode.TcpPort)
		if receivedCommand.Value == "" {
			receivedCommand = SendTCPRequest(byteMsg, replicaNode.Name+replicaNode.TcpPort)
			if receivedCommand.Value == "" {
				http.Error(w, fmt.Errorf("No such value present with key %s", receivedCommand.Key).Error(), http.StatusNotFound)
				return
			}
			value = []byte(receivedCommand.Value)
		} else {
			value = []byte(receivedCommand.Value)
		}
	}

	response := struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{
		Key:   key,
		Value: string(value),
	}
	byteRsp, err := json.Marshal(response)
	if err != nil {
		log.Fatal(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(byteRsp)
}

func SetObject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key, ok := vars["key"]
	if !ok {
		http.Error(w, errors.New("wrong query name").Error(), http.StatusBadRequest)
		return
	}
	value, ok := vars["value"]

	peersCommand := Command{
		Key:         key,
		Value:       value,
		CommandType: SET,
	}
	byteMsg, err := json.Marshal(peersCommand)
	if err != nil {
		log.Fatal(err)
	}

	originalNode, replicaNode := GetShardAndReplica(key)

	if originalNode.Id == GetNode().Id {
		store.NodeDataStore.SetValue(key, []byte(value))
		SendTCPRequest(byteMsg, replicaNode.Name+replicaNode.TcpPort)
	} else if replicaNode.Id == GetNode().Id {
		store.NodeDataStore.SetValue(key, []byte(value))
		SendTCPRequest(byteMsg, originalNode.Name+originalNode.TcpPort)
	} else {
		SendTCPRequest(byteMsg, originalNode.Name+originalNode.TcpPort)
		SendTCPRequest(byteMsg, replicaNode.Name+replicaNode.TcpPort)
	}

	store.NodeDataStore.PrintStoreContent()

	response := struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{
		Key:   key,
		Value: value,
	}
	byteMsg, _ = json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(byteMsg)

}
