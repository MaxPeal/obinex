package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Path: %s\n", r.URL.Path[1:])
	f, err := os.Open("../octopos-jenkins/testcase.microbench")
	if err != nil {
		//todo
		panic(err)
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	if err != nil {
		//todo
		panic(err)
	}
	log.Printf("served\n")
}

func main() {
	http.HandleFunc("/", handler)
	log.Println("running")
	http.ListenAndServe(":12334", nil)
}
