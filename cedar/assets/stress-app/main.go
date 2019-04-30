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
	requestRate, err := strconv.ParseFloat(os.Getenv("REQUESTS_PER_SECOND"), 64)
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
	responseSize, err := strconv.Atoi(os.Getenv("RESPONSE_SIZE"))
	responseFile, err2 := ioutil.TempFile("", "*")
	if err == nil && err2 == nil {
		// close responseFile on exit and check for its returned error
		defer func() {
			if err := responseFile.Close(); err != nil {
				log.Fatal(err)
			}
		}()

		token := make([]byte, 1024)
		for i := 0; i < (responseSize / 1024); i++ {
			if _, err := rand.Read(token); err != nil {
				log.Fatal(err)
				responseSize = 0
				break
			}

			// write a chunk
			if _, err := responseFile.Write(token); err != nil {
				log.Fatal(err)
				responseSize = 0
				break
			}
		}
	} else {
		responseSize = 0
	}
	minRequestDuration, err := strconv.Atoi(os.Getenv("MIN_REQUEST_DURATION"))
	if err != nil {
		minRequestDuration = 0
	}
	maxRequestDuration, err := strconv.Atoi(os.Getenv("MAX_REQUEST_DURATION"))
	if err != nil {
		maxRequestDuration = 0

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
		if minRequestDuration > 0 && maxRequestDuration > 0 {
			responseTime := rand.Intn(maxRequestDuration-minRequestDuration) + minRequestDuration
			time.Sleep(time.Duration(responseTime) * time.Second)
		}

		if responseSize > 0 {
			http.ServeFile(rw, r, responseFile.Name())
		} else {
			rw.Write(vcapApplicationBytes)
		}
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

	_, err = strconv.Atoi(os.Getenv("RESPONSE_SIZE"))
	if err != nil {
		fmt.Fprintf(os.Stdout, "%v\n", string(body))
	} else {
		fmt.Fprintf(os.Stdout, "received %d bytes\n", len(body))
	}
}
