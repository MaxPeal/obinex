package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	o "github.com/maxpeal/obinex"
	"golang.org/x/net/websocket"
)

// Channels for weblog output via websocket
var (
	wsChan    = make(chan o.WebData)
	wsAddChan = Broadcast(wsChan)
)

// persistentWebData holds status information that needs to be resent everytime
// the web view is updated.
var persistentWebData o.WebData

// Broadcast enables multiple reads from a channel.
// Subscribe by sending a channel into the returned Channel. The subscribed
// channel will now receive all messages sent into the original channel (c).
// TODO: see if possible to make generic for common.go
func Broadcast(c chan o.WebData) chan<- chan o.WebData {
	cNewChans := make(chan chan o.WebData)
	go func() {
		var cs []chan o.WebData
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

	type host struct {
		Boxname string
		Port    string
		Active  string
	}
	data := struct {
		Port  string
		Hosts []host
	}{
		Port: o.PortByBox[Boxname],
	}

	for _, box := range o.Boxes {
		active := ""
		if box == Boxname {
			active = "active"
		}
		data.Hosts = append(
			data.Hosts,
			host{
				Boxname: box,
				Port:    o.PortByBox[box],
				Active:  active,
			},
		)
	}

	err = t.Execute(w, data)
	if err != nil {
		fmt.Fprint(w, err)
	}
}

// websocketHandler sends log data to the javascript website
func websocketHandler(ws *websocket.Conn) {
	log.Printf("Web: connection to websocket")
	// show initial information on website
	websocket.JSON.Send(ws, persistentWebData)
	// give the channel some buffer to avoid message loss (see also comment
	// about blocking in Broadcast
	c := make(chan o.WebData, 10)
	wsAddChan <- c
	for {
		wd := <-c
		websocket.JSON.Send(ws, wd)
	}
	ws.Close()
}
