package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/lager/chug"
)

type Metric struct {
	Name      string
	Tags      map[string]string
	Value     string
	Timestamp time.Time
}

type Mapper struct {
	Name string

	StartString string
	EndString   string
	Transform   func(s, e chug.Entry) Metric
	GetKey      func(entry chug.Entry) (string, error)

	entriesMap map[string]chug.Entry
	mutex      sync.Mutex

	metricsFound int64
}

func (m *Mapper) processEntry(entry chug.Entry, metrics chan<- Metric) {
	if strings.Contains(entry.Log.Message, m.StartString) {
		key, err := m.GetKey(entry)
		if err != nil {
			return
		}
		m.setEntry(key, &entry)
		return
	}

	if strings.Contains(entry.Log.Message, m.EndString) {
		key, err := m.GetKey(entry)
		if err != nil {
			return
		}
		startEntry := m.getEntry(key)
		if startEntry != nil {
			metrics <- m.Transform(*startEntry, entry)
			m.setEntry(key, nil)
			atomic.AddInt64(&m.metricsFound, 1)
		}
		return
	}
}

func (m *Mapper) setEntry(key string, entry *chug.Entry) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if entry == nil {
		delete(m.entriesMap, key)
	} else {
		m.entriesMap[key] = *entry
	}
}

func (m *Mapper) getEntry(key string) *chug.Entry {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if entry, ok := m.entriesMap[key]; ok {
		return &entry
	}
	return nil
}

func mapAll(in <-chan chug.Entry, metrics chan<- Metric, mappers ...*Mapper) {
	batchSize := 1000
	semaphore := make(chan struct{}, batchSize)
	wg := sync.WaitGroup{}

	for entry := range in {
		semaphore <- struct{}{}
		go func(entry chug.Entry) {
			for _, mapper := range mappers {
				wg.Add(1)
				mapper.processEntry(entry, metrics)
				wg.Done()
			}
			<-semaphore
		}(entry)
	}
	wg.Wait()

	for _, mapper := range mappers {
		if mapper.metricsFound == 0 {
			fmt.Fprintf(os.Stderr, "no metrics found for mapper: %s\n", mapper.Name)
		} else {
			fmt.Fprintf(os.Stderr, "found %d metrics for mapper: %s\n", mapper.metricsFound, mapper.Name)
		}
	}
}
