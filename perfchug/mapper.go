package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/pivotal-golang/lager/chug"
)

type mapper struct {
	file io.Reader
}

func NewMapper(file io.Reader) mapper {
	return mapper{
		file: file,
	}
}

func logMapper(in <-chan chug.Entry) {
}

func (m mapper) mapper(entry chug.Entry) {
	mapper := make(map[string]chug.Entry)
	for entry := range in {
		key, err := getKey(entry)
		if err != nil {
			continue
		}

		if strings.Contains(entry.Log.Message, "request.serving") {
			mapper[key] = entry
		}

		if strings.Contains(entry.Log.Message, "request.done") {
			if servingEntry, ok := mapper[key]; ok {
				timeDiff := entry.Log.Timestamp.Sub(servingEntry.Log.Timestamp)
				fmt.Printf("%s: %s \n\n", key, timeDiff)
			}
		}
	}
}
