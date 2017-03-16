package obinex

import (
	"log"
	"os"
)

// DirecotryDateFormat is the format string used for timestamps in binary
// directries
const DirectoryDateFormat = "2006_01_02_-_15_04_05.000000000"

// WatchDir is the directory watched by obinex
// It must be absolute or relative to both obinex-server and obinex-watcher.
const WatchDir = "/proj/i4obinex/"

// SerialPath is the location of the serial connection
const SerialPath = "/dev/ttyS0"

// EndMarker is used to find the end of hw output
const EndMarker = "Graceful shutdown initiated"

// ControlHosts contains the mapping of buddy hostname to hardware box hostname.
var ControlHosts map[string]string = map[string]string{
	"localhost":       "mock",
	"faui49jenkins12": "faui49big01",
	"faui49jenkins13": "faui49big02",
	"faui49jenkins14": "faui49big03",
	"faui49jenkins15": "fastbox",
	"faui49jenkins21": "faui49jenkins25",
}

var HostByBox map[string]string = make(map[string]string)

func init() {
	for host, box := range ControlHosts {
		HostByBox[box] = host
	}
}

// Servers lists the servers connected to by default
var Servers = []string{
	"faui49jenkins12",
	"faui49jenkins13",
	"faui49jenkins14",
	"faui49jenkins15",
	"faui49jenkins21",
}

// BoxByHost returns the hardware box corresponding to a specific host
func BoxByHost(hostname string) (box string) {
	box, ok := ControlHosts[hostname]
	if !ok {
		box = "mock"
	}
	return
}

// CurrentBox returns the hardware box corresponding to the current host
func CurrentBox() (box string) {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	box = BoxByHost(hostname)
	return
}
