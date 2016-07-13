package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pivotal-golang/lager/chug"
)

var inputFilePath = flag.String(
	"inputFilePath",
	"",
	"Path to the input log file",
)

var outputFilePath = flag.String(
	"outputFilePath",
	"output.data",
	"Path to the output log file",
)

func main() {
	flag.Parse()

	file, err := os.Open(*inputFilePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	chugOut := make(chan chug.Entry)
	go chug.Chug(file, chugOut)
	logMapper(chugOut)
}

func getKey(entry chug.Entry) (string, error) {
	if entry.Log.Data["request"] == nil {
		return "", fmt.Errorf("not an http request")
	}
	return fmt.Sprintf("%s:%s", entry.Log.Data["request"], entry.Log.Session), nil
}

func writeData(name string, tags []string, value float64, timestamp int64) {
}
