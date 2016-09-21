package main

import (
	"bufio"
	"fmt"
	"html/template"
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

// Channels for synchronizing Run calls
var (
	binChan    = make(chan string)
	outputChan = make(chan string)
)

// Channel for weblog output via websocket
var wsChan = make(chan string)
var wsAddChan = o.Broadcast(wsChan)

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

// logHandler serves the website to view the logfile.
func logHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("weblog.html")
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Fprint(w, err)
		return
	}
	data := struct {
		Hostname    string
		HardwareBox string
	}{
		Hostname:    hostname,
		HardwareBox: o.ControlHosts[hostname],
	}
	err = t.Execute(w, data)
	if err != nil {
		fmt.Fprint(w, err)
	}
}

// logWebsocket sends log data to the javascript website
func logWebsocket(ws *websocket.Conn) {
	c := make(chan string)
	wsAddChan <- c
	for {
		line := <-c
		fmt.Fprintf(ws, line)
	}
	ws.Close()
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
		parseLine := strings.TrimSpace(line)
		if parseLine == "Graceful shutdown initiated" ||
			strings.HasPrefix(parseLine, "Could not boot") {
			outputChan <- s
			s = ""
		}
		wsChan <- line
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
