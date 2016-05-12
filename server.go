package main

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"os"
)

import (
	"github.com/tarm/serial"
)

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Server: requested path: %s\n", r.URL.Path[1:])
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
	log.Printf("Server: binary served\n")
}

func getOutput(c chan string) {
	conf := &serial.Config{
		Name:   "/dev/ttyS0",
		Baud:   115200,
		Parity: serial.ParityNone,
		Size:   8,
	}
	s, err := serial.OpenPort(conf)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	r := bufio.NewReader(s)
	for err == nil {
		var line string
		line, err = r.ReadString('\n')
		c <- line
	}
	if err != nil {
		log.Fatal(err)
	}
}

func handleOutput(c chan string) {
	for line := range c {
		log.Printf("Output: %s", line)
	}
}

func main() {
	http.HandleFunc("/", handler)
	log.Println("Server: running")
	c := make(chan string, 10)
	go getOutput(c)
	go handleOutput(c)
	http.ListenAndServe(":12334", nil)
}
