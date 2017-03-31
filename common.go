package obinex

import (
	"crypto/md5"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type WorkPackage struct {
	Path     string
	Checksum [md5.Size]byte
}

func changeStateOnPath(path, state string) string {
	n := len(WatchDir)
	if WatchDir == "./" {
		n = 0
	}
	// Split into box, state and rest of path
	parts := strings.SplitN(path[n:], string(filepath.Separator), 3)
	box := parts[0]
	path = parts[2]
	path = filepath.Join(WatchDir, box, state, path)
	return path
}

func (wp *WorkPackage) ToQueued() error {
	// Set the checksum
	f, err := os.Open(wp.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := md5.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return err
	}
	wp.Checksum = md5.Sum(nil)

	// Create new structure
	bin := wp.Path
	org := bin
	t := time.Now().Format(DirectoryDateFormat)
	dir := filepath.Dir(bin) + "/"
	bin = filepath.Base(bin)
	dir = changeStateOnPath(dir, "queued")
	dir = filepath.Join(dir, bin+"_"+t)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		log.Println("Mkdir Error:", err)
		return err
	}

	// Move bin
	err = os.Rename(org, filepath.Join(dir, bin))
	if err != nil {
		log.Println("Rename Error:", err)
		return err
	}
	wp.Path = filepath.Join(dir, bin)
	return nil
}

func (wp *WorkPackage) toY(y string) {
	dir := filepath.Dir(wp.Path) + "/"
	new := changeStateOnPath(dir, y)
	err := os.MkdirAll(filepath.Join(new, ".."), 0755)
	if err != nil {
		log.Println("Output Error:", err)
		return
	}

	// Move dir
	err = os.Rename(dir, new)
	if err != nil {
		log.Println("Output Error:", err)
	}
	wp.Path = filepath.Join(new, filepath.Base(wp.Path))
}

func (wp *WorkPackage) ToExecuting() {
	wp.toY("executing")
}

func (wp *WorkPackage) ToOut() {
	wp.toY("out")
}

// DirecotryDateFormat is the format string used for timestamps in binary
// directries
const DirectoryDateFormat = "2006_01_02_-_15_04_05.000000000"

// WatchDir is the directory watched by obinex
// It must be absolute or relative to both obinex-server and obinex-watcher.
var WatchDir = "/proj/i4obinex/"

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
