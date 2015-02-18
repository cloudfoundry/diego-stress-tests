package main

import (
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	vcap_application := os.Getenv("VCAP_APPLICATION")
	vcap_application_bytes := []byte(vcap_application)

	serveErrChan := make(chan error)
	timer := time.NewTimer(30 * time.Second)

	go func() {
		serveErrChan <- http.ListenAndServe("0.0.0.0:"+os.Getenv("PORT"), http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			log.Printf("serving request: %#v\n", r)
			rw.Write(vcap_application_bytes)
		}))
	}()

	select {
	case err := <-serveErrChan:
		if err != nil {
			log.Fatalln(err.Error())
		}
	case <-timer.C:
		log.Fatalln("Crashing")
	}
}
