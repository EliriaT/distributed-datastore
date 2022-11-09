package cluster

import (
	"encoding/json"
	"errors"
	"github.com/EliriaT/distributed-datastore/store"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func GetObject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key, ok := vars["key"]
	if !ok {
		http.Error(w, errors.New("wrong query name").Error(), http.StatusBadRequest)
		return
	}
	value, err := store.NodeDataStore.GetValue(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
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

	store.NodeDataStore.SetValue(key, []byte(value))

	store.NodeDataStore.PrintStoreContent()

	peersCommand := Command{
		Key:         key,
		Value:       value,
		CommandType: SET,
	}
	SendCommandToPeers(peersCommand)

	response := struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}{
		Key:   key,
		Value: value,
	}
	byteMsg, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(byteMsg)

}
