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
	"os/exec"
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
	persistentWebData.Queue = append(persistentWebData.Queue, wp.Path[len(o.WatchDir)+len(Boxname)+11:])
	wsChan <- persistentWebData
	runToServChan <- wp
	<-eoeChan
	log.Printf("RPC: binary request return: %s\n", wp.Path)
	return nil
}

func (r *Rpc) Powercycle(_ struct{}, output *string) error {
	log.Printf("RPC: powercycle\n")
	cmd := exec.Command("bash", "-c", o.PowercyclePath, Boxname)
	outputRaw, err := cmd.CombinedOutput()
	*output = string(outputRaw)
	return err
}

// UpdateWebView allows obinex-watcher to send data to the web status page.
func (r *Rpc) UpdateWebView(wd o.WebData, _ *struct{}) error {
	persistentWebData.Lock = wd.Lock
	if wd.Mode != "" {
		persistentWebData.Mode = wd.Mode
	}
	wsChan <- persistentWebData
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
	for {
		i -= 1
		if i == -1 {
			break
		}
		if err != nil {
			log.Println(err)
			log.Println("Server: checksum calculation failed, retrying...")
		} else {
			ok := true
			checksum := h.Sum(nil)
			if len(checksum) == len(wp.Checksum) {
				for i, e := range checksum {
					if e != wp.Checksum[i] {
						ok = false
						break
					}
				}
			} else {
				ok = false
				log.Printf("Server: checksum sizes don't match, this should not happen\n")
			}
			if ok {
				break
			}
			log.Printf("Server: checksum doesn't match, retrying...\n")
		}
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
		log.Printf("Server: error (ignored): %s\n", err)
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
		if len(persistentWebData.Queue) > 0 {
			persistentWebData.Queue = persistentWebData.Queue[1:]
		}
		wsChan <- persistentWebData
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
			wsChan <- o.WebData{LogLine: line}
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

func updateBootMode() {
	for {
		output, err := o.ExecCommand("nc", "-w", "2", "-zv", Boxname, "22")
		log.Println("Bootmode:", string(output))
		if err == nil {
			persistentWebData.Mode = "batch"
		} else {
			persistentWebData.Mode = "linux"
		}
		wsChan <- persistentWebData
		time.Sleep(30 * time.Second)
	}
}

func main() {
	flag.Parse()

	http.HandleFunc("/"+Boxname, binaryServeHandler)
	http.HandleFunc("/", weblogHandler)
	http.Handle("/logws", websocket.Handler(websocketHandler))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))
	c := make(chan string, 10)
	go getSerialOutput(c)
	go handleOutput(c)

	//go updateBootMode()

	rpc.Register(new(Rpc))
	rpc.HandleHTTP()

	server := &http.Server{
		Addr: o.PortByBox[Boxname],
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			addr := req.RemoteAddr
			// allow the following addrs to access every path
			if strings.HasPrefix(addr, "131.188.42.") ||
				strings.HasPrefix(addr, "[::1]") ||
				strings.HasPrefix(addr, "127.0.0.1") ||
				// allow these paths from everywhere
				req.URL.Path == "/" ||
				req.URL.Path == "/logws" ||
				strings.HasPrefix(req.URL.Path, "/static/") {

				http.DefaultServeMux.ServeHTTP(w, req)
				return
			}
			http.Error(w, "", 404)
			log.Printf("Blocked access from %s\n", addr)
		}),
	}
	log.Printf("Server: %s serving %s\n", server.Addr, Boxname)
	log.Println(graceful.ListenAndServe(server, 10*time.Second))
}
