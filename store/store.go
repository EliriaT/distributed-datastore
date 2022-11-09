package store

import (
	"fmt"
	"log"
)

var NodeDataStore = NewStore()

type Store struct {
	dataStore map[string][]byte
}

func (s *Store) GetValue(key string) ([]byte, error) {
	value, ok := s.dataStore[key]

	if !ok {
		err := fmt.Errorf("No such value present with key %s", key)
		return nil, err
	}

	return value, nil
}

func (s *Store) SetValue(key string, value []byte) {
	s.dataStore[key] = value
}

func (s *Store) PrintStoreContent() {
	log.Println("---------------------------------------")
	log.Println(s.dataStore)
	log.Println("---------------------------------------")
}

func NewStore() Store {
	return Store{
		dataStore: make(map[string][]byte),
	}
}
