package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	endpointToHit := os.Getenv("ENDPOINT_TO_HIT")
	logRate, err := strconv.ParseFloat(os.Getenv("LOGS_PER_SECOND"), 64)
	if err != nil {
		log.Fatal(err)
	}
	requestRate, err := strconv.ParseFloat(os.Getenv("REQUEST_PER_SECOND"), 64)
	if err != nil {
		log.Fatal(err)
	}
	minSecondsTilCrash, err := strconv.Atoi(os.Getenv("MIN_SECONDS_TIL_CRASH"))
	if err != nil {
		minSecondsTilCrash = 0
	}
	maxSecondsTilCrash, err := strconv.Atoi(os.Getenv("MAX_SECONDS_TIL_CRASH"))
	if err != nil {
		maxSecondsTilCrash = 0
	}

	vcapApplication := os.Getenv("VCAP_APPLICATION")
	vcapApplicationBytes := []byte(vcapApplication)

	var requestTicker, logTicker *time.Ticker
	var crashTimer *time.Timer

	if requestRate > 0 {
		requestTicker = time.NewTicker(time.Duration(float64(time.Second) / requestRate))
	} else {
		requestTicker = time.NewTicker(time.Hour)
		requestTicker.Stop()
	}

	if logRate > 0 {
		logTicker = time.NewTicker(time.Duration(float64(time.Second) / logRate))
	} else {
		logTicker = time.NewTicker(time.Hour)
		logTicker.Stop()
	}

	rand.Seed(int64(time.Now().Nanosecond()))

	if minSecondsTilCrash > 0 && maxSecondsTilCrash > 0 {
		secondsTilCrash := rand.Intn(maxSecondsTilCrash-minSecondsTilCrash) + minSecondsTilCrash
		log.Printf("Crashing in %d seconds\n", secondsTilCrash)
		crashTimer = time.NewTimer(time.Second * time.Duration(secondsTilCrash))
	} else {
		crashTimer = time.NewTimer(time.Hour)
		crashTimer.Stop()
	}

	go func() {
		for {
			select {
			case <-requestTicker.C:
				go hitEndpoint(endpointToHit)
			case <-logTicker.C:
				go log.Println(vcapApplication)
			case <-crashTimer.C:
				panic("freak out")
			}
		}
	}()

	err = http.ListenAndServe("0.0.0.0:"+os.Getenv("PORT"), http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Write(vcapApplicationBytes)
	}))

	if err != nil {
		log.Fatal(err)
	}
}

func hitEndpoint(endpoint string) {
	resp, err := http.Get(endpoint)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}
	fmt.Fprintf(os.Stdout, "%v\n", string(body))
}
