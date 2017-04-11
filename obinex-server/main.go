package main

import (
	"bufio"
	"crypto/md5"
	"flag"
	"io"
	"log"
	"net/http"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tarm/serial"
	"golang.org/x/net/websocket"
	"gopkg.in/tylerb/graceful.v1"

	o "gitlab.cs.fau.de/luksen/obinex"
)

// testDone is used by the testsuite
var testDone = make(chan bool, 1)

// binQueue lists all queued binaries.
// This is non-functional purely for logging etc.
var binQueue []string

// Channels for synchronizing Run calls
var (
	runToServChan = make(chan o.WorkPackage)
	servToOutChan = make(chan o.WorkPackage)
	eoeChan       = make(chan struct{}) // eoe = end of execution
	lateEoeChan   = make(chan struct{})
)

// Rpc provides the public methods needed for rpc.
type Rpc struct{}

// Run allows a remote caller to request execution of a binary.
// The Path should be absolute or relative to the _server_ binary.
func (r *Rpc) Run(wp o.WorkPackage, _ *struct{}) error {
	log.Printf("RPC: binary request: %s\n", wp.Path)
	boxname := o.CurrentBox()
	binQueue = append(binQueue, wp.Path[len(o.WatchDir)+len(boxname)+4:])
	wsChan <- WebData{Queue: binQueue}
	runToServChan <- wp
	<-eoeChan
	log.Printf("RPC: binary request return: %s\n", wp.Path)
	return nil
}

// binaryServeHandler serves the binaries to the hardware.
func binaryServeHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Server: binary requested by hardware\n")
	lateEoeChan <- struct{}{}
	wp := <-runToServChan

	// Sometimes there is a delay before we can access the file via NFS, so
	// we wait up to a second before erroring out.
	f, err := os.Open(wp.Path)
	i := 10
	for err != nil {
		i -= 1
		if i == -1 {
			break
		}
		log.Printf("Server: file access problem, retrying...\n")
		time.Sleep(100 * time.Millisecond)
		f, err = os.Open(wp.Path)
	}
	if err != nil {
		log.Println(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// To make sure the file doesn't only exist but is also complete, we
	// compare checksums as well.
	h := md5.New()
	_, err = io.Copy(h, f)
	i = 10
	for err != nil {
		i -= 1
		if i == -1 {
			break
		}
		log.Printf("Server: checksum doesn't match, retrying...\n")
		time.Sleep(100 * time.Millisecond)
		f.Close()
		f, _ = os.Open(wp.Path)
		h = md5.New()
		_, err = io.Copy(h, f)
	}
	defer f.Close()
	f.Seek(0, 0)

	// This is for handleOutput. We do this here because we can be sure
	// that there was an rpc-request as well as an http-request. Also
	// lateEoe has been signalled, so the old output is definitley done.
	servToOutChan <- wp

	n, err := io.Copy(w, f)
	log.Printf("Server: served %dbytes", n)
	if err != nil {
		panic(err)
	}
	log.Printf("Server: binary served\n")
}

// getSerialOutput handles the serial communication with the hardware.
// The output is sent line by line to the provided channel.
func getSerialOutput(c chan string) {
	conf := &serial.Config{
		Name:   SerialPath,
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
		select {
		case <-testDone:
			log.Println("Test: exiting getSerialOutput")
			return
		default:
			break
		}
	}
	if err != nil {
		log.Println("Getting serial output: ", err)
	}
}

// handleOutput takes output from the provided channel and distributes it.
func handleOutput(c chan string) {
	runningBin := false
	var f *os.File
	var err error
	endOfBin := func() {
		if len(binQueue) > 0 {
			binQueue = binQueue[1:]
		}
		wsChan <- WebData{Queue: binQueue}
		eoeChan <- struct{}{}
		if f != nil {
			f.Close()
		}
		runningBin = false
	}
	for {
		select {
		case line := <-c:
			if runningBin {
				if f != nil {
					f.WriteString(line)
				}
			}
			parseLine := strings.TrimSpace(line)
			wsChan <- WebData{LogLine: line}
			// detect end of execution early
			if strings.HasPrefix(parseLine, o.EndMarker) ||
				strings.HasPrefix(parseLine, "Could not boot") {
				endOfBin()
			}
		case <-lateEoeChan:
			// if no end of execution was detected
			if runningBin {
				endOfBin()
			}
		case wp := <-servToOutChan:
			f, err = os.Create(filepath.Join(filepath.Dir(wp.Path), "output.txt"))
			if err != nil {
				log.Println("Server:", err)
			}
			runningBin = true
			log.Println("Server: executing", wp.Path)
		case <-testDone:
			log.Println("Test: exiting handleOutput")
			return
		}
	}
}

func main() {
	flag.Parse()

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	box := o.BoxByHost(hostname)
	http.HandleFunc("/"+box, binaryServeHandler)
	http.HandleFunc("/", weblogHandler)
	http.Handle("/logws", websocket.Handler(websocketHandler))
	c := make(chan string, 10)
	go getSerialOutput(c)
	go handleOutput(c)

	rpc.Register(new(Rpc))
	rpc.HandleHTTP()

	server := &http.Server{
		Addr: ":12334",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			addr := req.RemoteAddr
			if !(strings.HasPrefix(addr, "[::1]") || strings.HasPrefix(addr, "127.0.0.1") || strings.HasPrefix(addr, "131.188.42.")) {
				http.Error(w, "Blocked", 401)
				log.Printf("Blocked access from %s\n", addr)
				return
			}
			http.DefaultServeMux.ServeHTTP(w, req)
		}),
	}
	log.Printf("Server: %s serving %s\n", hostname, box)
	log.Println(graceful.ListenAndServe(server, 10*time.Second))
}
