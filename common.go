package obinex

import (
	"crypto/md5"
	"io"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// WebData should be used to send data via the websocket
type WebData struct {
	LogLine string
	Queue   []string
	Lock    string
	Mode    string
}

type WorkPackage struct {
	Path     string
	Checksum []byte
}

// RpcArg encapsulates rpc arguments from the clt to obinex-watcher
type RpcArg struct {
	Boxname string
	Uid     uint32
}

// ExecCommand is a simple wrapper around a common usage of exec.Command.
// Having this in a separate function also allows us to mock this function for
// testing.
var ExecCommand = func(cmd string, args ...string) (output []byte, err error) {
	c := exec.Command(cmd, args...)
	output, err = c.CombinedOutput()
	return
}

// Username returns the human readable username for a uid
func Username(uid uint32) string {
	username := "unknown"
	u, err := user.LookupId(strconv.Itoa(int(uid)))
	if err == nil {
		username = u.Username
	} else {
		log.Println("Couldn't get username:")
		log.Println(err)
	}
	return username
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
	wp.Checksum = h.Sum(nil)

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

// WatcherHost tells us where obinex-watcher is running
var WatcherHost = "i4jenkins"

// PowercyclePath is the location of the powercycle script
const PowercyclePath = "/proj/i4invasic/bin/powerCycle.sh"

// BootModePath points to the script that changes boot mode
const BootModePath = "/proj/i4invasic/tftpboot/switchboot.pl"

// DirecotryDateFormat is the format string used for timestamps in binary
// directries
const DirectoryDateFormat = "2006_01_02_-_15_04_05.000000000"

// WatchDir is the directory watched by obinex
// It must be absolute or relative to both obinex-server and obinex-watcher.
var WatchDir = "/proj/i4obinex/"

// SerialPath is the location of the serial connection
const SerialPath = "/dev/ttyS0"

// EndMarker is used to find the end of hw output
const EndMarker = "octopos-shutdown "

// PortByBox maps Boxnames to their webserver port
var PortByBox map[string]string = map[string]string{
	"mock":        ":12230",
	"faui49big01": ":12231",
	"faui49big02": ":12232",
	"faui49big03": ":12233",
	"fastbox":     ":12234",
}

// StringList can be used for list-like command line arguments
type StringList []string

func (sl *StringList) String() string {
	return strings.Join(*sl, ",")
}

func (sl *StringList) Set(value string) error {
	*sl = StringList(strings.Split(value, ","))
	return nil
}
