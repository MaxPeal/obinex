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
	o "gitlab.cs.fau.de/luksen/obinex"
	"golang.org/x/net/websocket"
)

// binQueue lists all queued binaries.
// This is non-functional purely for logging etc.
var binQueue []string

// Channels for synchronizing Run calls
var (
	binChan            = make(chan string)
	outputChan         = make(chan string)
	activateOutputChan = make(chan struct{})
)

// Rpc provides the public methods needed for rpc.
type Rpc struct{}

// Run allows a remote caller to request execution of a binary.
// The Path should be absolute or relative to the _server_ binary.
func (r *Rpc) Run(path string, reply *string) error {
	log.Printf("RPC: binary request: %s\n", path)
	var boxname string
	hostname, err := os.Hostname()
	if err == nil {
		boxname = o.ControlHosts[hostname]
	}
	binQueue = append(binQueue, path[len(o.WatchDir)+len(boxname)+4:])
	wsChan <- WebData{Queue: binQueue}
	binChan <- path
	*reply = <-outputChan
	log.Printf("RPC: binary request return: %s\n", path)
	return nil
}

// binaryServeHandler serves the binaries to the hardware.
func binaryServeHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Server: binary requested\n")
	activateOutputChan <- struct{}{}
	bin := <-binChan
	f, err := os.Open(bin)
	if err != nil {
		log.Println("Server: ", err)
		return
	}
	defer f.Close()
	_, err = io.Copy(w, f)
	if err != nil {
		log.Println("Server: ", err)
		return
	}
	log.Printf("Server: binary served\n")
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
	runningBin := false
	endOfBin := func() {
		log.Printf("Server: end of binary output\n")
		binQueue = binQueue[1:]
		wsChan <- WebData{Queue: binQueue}
		outputChan <- s
		s = ""
		runningBin = false
	}
	for {
		select {
		case line := <-c:
			log.Printf("Output: %s", line)
			if runningBin {
				s += line
			}
			parseLine := strings.TrimSpace(line)
			wsChan <- WebData{LogLine: line}
			// detect end of execution early
			if parseLine == "Graceful shutdown initiated" ||
				strings.HasPrefix(parseLine, "Could not boot") {
				endOfBin()
			}
		case <-activateOutputChan:
			// if no end of execution was detected
			if runningBin {
				endOfBin()
			}
			runningBin = true
			log.Printf("Server: start of binary output\n")
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
	http.HandleFunc("/"+o.ControlHosts[hostname], binaryServeHandler)
	http.HandleFunc("/", logHandler)
	log.Printf("Server: %s serving %s\n", hostname, o.ControlHosts[hostname])
	http.Handle("/logws", websocket.Handler(logWebsocket))
	c := make(chan string, 10)
	go getOutput(c)
	go handleOutput(c)

	rpc.Register(new(Rpc))
	rpc.HandleHTTP()

	http.ListenAndServe(":12334", nil)
}
