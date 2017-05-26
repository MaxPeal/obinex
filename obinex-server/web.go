package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

import (
	o "gitlab.cs.fau.de/luksen/obinex"
	"golang.org/x/net/websocket"
)

// WebData should be used to send data via the websocket
type WebData struct {
	LogLine string
	Queue   []string
}

// Channels for weblog output via websocket
var (
	wsChan    = make(chan WebData)
	wsAddChan = Broadcast(wsChan)
)

// Broadcast enables multiple reads from a channel.
// Subscribe by sending a channel into the returned Channel. The subscribed
// channel will now receive all messages sent into the original channel (c).
// TODO: see if possible to make generic for common.go
func Broadcast(c chan WebData) chan<- chan WebData {
	cNewChans := make(chan chan WebData)
	go func() {
		var cs []chan WebData
		for {
			select {
			case newChan := <-cNewChans:
				cs = append(cs, newChan)
			case e := <-c:
				for _, outC := range cs {
					// send non-blocking to avoid one
					// channel breaking the whole
					// broadcast
					select {
					case outC <- e:
						break
					default:
						break
					}
				}
			}
		}
	}()
	return cNewChans
}

// weblogHandler serves the website to view the logfile.
func weblogHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("web/base.html", "web/status.html")
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
		Hostname:    hostname + ".informatik.uni-erlangen.de",
		HardwareBox: o.BoxByHost(hostname),
	}
	if data.HardwareBox == "mock" {
		data.Hostname = "localhost"
	}
	err = t.Execute(w, data)
	if err != nil {
		fmt.Fprint(w, err)
	}
}

// websocketHandler sends log data to the javascript website
func websocketHandler(ws *websocket.Conn) {
	log.Printf("Web: connection to websocket")
	// immediately show queue on website
	websocket.JSON.Send(ws, WebData{Queue: binQueue})
	// give the channel some buffer to avoid message loss (see also comment
	// about blocking in Broadcast
	c := make(chan WebData, 10)
	wsAddChan <- c
	for {
		wd := <-c
		websocket.JSON.Send(ws, wd)
	}
	ws.Close()
}
