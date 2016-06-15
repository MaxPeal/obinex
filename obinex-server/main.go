package main

import (
	"bufio"
	"fmt"
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

// controlHosts contains the mapping of buddy hostname to hardware box hostname.
var controlHosts map[string]string = map[string]string{
	"faui49jenkins12": "faui49big01",
	"faui49jenkins13": "faui49big02",
	"faui49jenkins14": "faui49big03",
	"faui49jenkins21": "faui49jenkins25",
	"faui49bello2":    "fastbox",
}

// Channels for synchronizing Run calls
var (
	binChan    = make(chan string)
	outputChan = make(chan string)
)

// Rpc provides the public methods needed for rpc.
type Rpc struct{}

// Run allows a remote caller to request execution of a binary.
// The Path should be absolute or relative to the _server_ binary.
func (r *Rpc) Run(path string, reply *string) error {
	log.Printf("RPC: binary request: %s\n", path)
	binChan <- path
	*reply = <-outputChan
	return nil
}

// binaryServeHandler serves the binaries to the hardware.
func binaryServeHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Server: binary requested\n")
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

// logHandler serves the logfile.
func logHandler(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open("obinex.log")
	if err != nil {
		fmt.Fprint(w, err)
	}
	fmt.Fprint(w, "<html><body><pre>")
	_, err = io.Copy(w, f)
	if err != nil {
		fmt.Fprint(w, err)
	}
	fmt.Fprint(w, "</pre></body></html>")
}

// getOutput handles the serial communication with the hardware.
// The output is sent line by line to the provided channel.
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

// handleOutput takes output from the provided channel and distributes it.
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
	// log to stdout and a file
	f, err := os.Create("obinex.log")
	if err != nil {
		log.Print("no log file:", err)
	} else {
		mw := io.MultiWriter(os.Stdout, f)
		log.SetOutput(mw)
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/"+controlHosts[hostname], binaryServeHandler)
	http.HandleFunc("/", logHandler)
	log.Printf("Server: %s serving %s\n", hostname, controlHosts[hostname])
	c := make(chan string, 10)
	go getOutput(c)
	go handleOutput(c)

	rpc.Register(new(Rpc))
	rpc.HandleHTTP()

	http.ListenAndServe(":12334", nil)
}
