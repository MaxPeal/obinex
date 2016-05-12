package main

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"net/rpc"
	"os"
	"strings"
)

import (
	"github.com/tarm/serial"
)

var binChan = make(chan string)
var outputChan = make(chan string)

type Rpc struct{}

func (r *Rpc) Run(bin string, reply *string) error {
	log.Printf("RPC: binary request: %s\n", bin)
	binChan <- bin
	*reply = <-outputChan
	return nil
}

func binaryServeHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Server: requested path: %s\n", r.URL.Path[1:])
	bin := <-binChan
	f, err := os.Open(bin)
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
	var s string
	for line := range c {
		log.Printf("Output: %s", line)
		s += line
		if strings.TrimSpace(line) == "Graceful shutdown initiated" {
			outputChan <- s
			s = ""
		}
	}
}

func main() {
	http.HandleFunc("/", binaryServeHandler)
	log.Println("Server: running")
	c := make(chan string, 10)
	go getOutput(c)
	go handleOutput(c)

	rpc.Register(new(Rpc))
	rpc.HandleHTTP()

	http.ListenAndServe(":12334", nil)
}
