package main

import (
	"flag"
	"os"
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
}
