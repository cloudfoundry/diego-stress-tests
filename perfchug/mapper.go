package main

import (
	"strings"
	"sync"
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
	StartString string
	EndString   string
	Transform   func(s, e chug.Entry) Metric
	GetKey      func(entry chug.Entry) (string, error)

	entriesMap map[string]chug.Entry
	s          sync.RWMutex
}

func (m *Mapper) processEntry(entry chug.Entry, metrics chan<- Metric) {
	// m.s.Lock()
	// defer m.s.Unlock()

	if strings.Contains(entry.Log.Message, m.StartString) {
		key, err := m.GetKey(entry)
		if err != nil {
			return
		}
		m.entriesMap[key] = entry
		return
	}

	if strings.Contains(entry.Log.Message, m.EndString) {
		key, err := m.GetKey(entry)
		if err != nil {
			return
		}
		if startEntry, ok := m.entriesMap[key]; ok {
			metrics <- m.Transform(startEntry, entry)
			delete(m.entriesMap, key)
		}
		return
	}
}

func mapAll(in <-chan chug.Entry, metrics chan<- Metric, mappers ...Mapper) {
	// wg := sync.WaitGroup{}
	for entry := range in {
		for _, mapper := range mappers {
			// wg.Add(1)
			// go func() {
			mapper.processEntry(entry, metrics)
			// wg.Done()
			// }()
		}
		// wg.Wait()
	}
}
