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

	peersCommand := DbCommand{
		Key:   key,
		Value: "",
	}
	toDoCommand := Command{
		Action:  GET,
		Payload: peersCommand}

	byteMsg, err := json.Marshal(toDoCommand)
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
			receivedCommand, err2 := SendTCPRequest(byteMsg, tcpAddres)
			// if there is an error, this means that the value can not be found
			if err2 != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			dbCommand := convertFromMapToDbCommand(receivedCommand.Payload)
			if dbCommand.Value == "" {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			value = []byte(dbCommand.Value)
		}
	} else {
		var dbCommand DbCommand
		receivedCommand, err := SendTCPRequest(byteMsg, originalNode.Name+originalNode.TcpPort)
		if err == nil {
			dbCommand = convertFromMapToDbCommand(receivedCommand.Payload)
		}

		if dbCommand.Value == "" {
			receivedCommand, err = SendTCPRequest(byteMsg, replicaNode.Name+replicaNode.TcpPort)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			dbCommand = convertFromMapToDbCommand(receivedCommand.Payload)
			if dbCommand.Value == "" {
				http.Error(w, fmt.Errorf("No such value present with key %s", dbCommand.Key).Error(), http.StatusNotFound)
				return
			}
			value = []byte(dbCommand.Value)
		} else {
			value = []byte(dbCommand.Value)
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

	peersCommand := DbCommand{
		Key:   key,
		Value: value,
	}
	toDoCommand := Command{
		Action:  SET,
		Payload: peersCommand}

	byteMsg, err := json.Marshal(toDoCommand)
	if err != nil {
		log.Fatal(err)
	}

	originalNode, replicaNode := GetShardAndReplica(key)
	GetConsensusModule().AddLogEntry(originalNode.Id, replicaNode.Id, peersCommand)

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
