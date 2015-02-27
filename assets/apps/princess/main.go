package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	vcap_application := os.Getenv("VCAP_APPLICATION")
	vcap_application_bytes := []byte(vcap_application)

	// go func() {
	// 	for {
	// 		log.Println(vcap_application)
	// 		time.Sleep(time.Second / 20)
	// 	}
	// }()

	err := http.ListenAndServe("0.0.0.0:"+os.Getenv("PORT"), http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		log.Printf("serving request: %#v\n", r)
		rw.Write(vcap_application_bytes)
	}))
	if err != nil {
		log.Fatalln(err.Error())
	}
}
