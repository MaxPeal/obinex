package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/luksen/fsnotify"
	o "gitlab.cs.fau.de/luksen/obinex"
)

type Buddy struct {
	Boxname    string
	Servername string
	InDir      string
	Lock       Lock
	ModePath   string
	queue      chan o.WorkPackage
	rpc        *rpc.Client
}

func (b *Buddy) Connect() error {
	log.Println("RPC: connecting to", b.Servername)
	client, err := rpc.DialHTTP("tcp", b.Servername+":12334")
	if err != nil {
		return err
	}
	b.rpc = client
	log.Printf("RPC: %s connected\n", b.Servername)
	return nil
}

func (b *Buddy) Run(wp o.WorkPackage) {
	if b.rpc == nil {
		log.Println("RPC: not connected yet")
		return
	}
	err := b.rpc.Call("Rpc.Run", wp, nil)
	if err != nil {
		log.Println("RPC:", err)
	}
	wp.ToOut()
}

func (b *Buddy) UpdateWebView(wd o.WebData) {
	if b.rpc == nil {
		log.Println("RPC: not connected yet")
		return
	}
	err := b.rpc.Call("Rpc.UpdateWebView", wd, nil)
	if err != nil {
		log.Println("RPC:", err)
	}
}

func (b *Buddy) Close() {
	b.rpc.Close()
}

func (b *Buddy) Enqueue(path string) {
	if b.Lock.Get(path) {
		log.Println("Watcher: queueing", path)
		wp := o.WorkPackage{Path: path}
		err := wp.ToQueued()
		if err != nil {
			log.Println("Error:", err)
			return
		}
		b.queue <- wp
		return
	}
	log.Println("Watcher: blocked", path)
}

func (b *Buddy) SetBootMode(mode string) {
	if mode != "linux" &&
		mode != "batch" &&
		mode != "nfs" &&
		mode != "interactive" {
		log.Printf("Invalid mode \"%s\". Mode not changed.\n", mode)
		return
	}
	log.Printf("Changing %s mode to \"%s\"\n", b.Boxname, mode)

	if !b.Lock.Get(b.ModePath) {
		log.Println("Mode change denied by lock. If you are the owner of the lock, try removing the mode file first.")
		return
	}

	cmd := exec.Command("bash", "-c", o.BootModePath+" "+b.Boxname+" "+mode)
	_, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("mode error:", err)
		return
	}
	b.UpdateWebView(o.WebData{Mode: mode})
	log.Println("Mode changed.")
}

func (b *Buddy) walkAndRun(dir string, watcher *fsnotify.Watcher) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println("Watcher:", err)
			return err
		}
		if path == b.Lock.Path {
			err := b.Lock.Set()
			return err
		}
		// Mode changes are only done via file event, not in the
		// initial walk since that would lead to unnecessary mode
		// changes.
		if path == b.ModePath {
			return nil
		}
		if info.IsDir() == false {
			b.Enqueue(path)
			return nil
		}

		// If no watcher is given, we only want to enqueue and lock
		if watcher == nil {
			return nil
		}

		err = watcher.Add(path)
		if err != nil {
			log.Println("Watcher: fsnotify error:", err)
			return nil
		}
		log.Println("Watcher: watching " + path)
		return nil
	})
}

func shouldRetry(err error) bool {
	if err.Error() == "connection is shut down" {
		return true
	}
	if err, ok := err.(*net.OpError); ok {
		if err, ok := err.Err.(*os.SyscallError); ok {
			if err.Err == syscall.ECONNREFUSED || err.Err == syscall.ESHUTDOWN {
				return true
			}
		}
	}
	return false
}

func retryWatchAndRun(buddy *Buddy, done chan bool) {
	defer func() { done <- true }()
	go buddy.RunQueue()
	err := watchAndRun(buddy)
	for shouldRetry(err) {
		log.Println("Connection error, retrying...")
		time.Sleep(1 * time.Second)
		err = watchAndRun(buddy)
	}
	log.Printf("watchAndRun [%s]: %s", buddy.Servername, err)
	return
}

func (b *Buddy) RunQueue() {
	for wp := range b.queue {
		wp.ToExecuting()
		go b.Run(wp)
	}
}

func watchAndRun(buddy *Buddy) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("Watcher:", err)
		return err
	}
	defer watcher.Close()

	os.MkdirAll(buddy.InDir, 0755)
	err = buddy.walkAndRun(buddy.InDir, watcher)
	if err != nil {
		log.Println(err)
	}

	killChan := make(chan os.Signal, 1)
	signal.Notify(killChan, syscall.SIGTERM)
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Remove == fsnotify.Remove {
				if event.Name == buddy.Lock.Path {
					buddy.Lock.Unset()
				}
			}
			if event.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(event.Name)
				if err != nil {
					log.Println("Watcher:", err)
					break
				}
				if info.IsDir() {
					err = buddy.walkAndRun(event.Name, watcher)
					if err != nil {
						log.Println(err)
					}
					break
				}
			}
			if event.Op&fsnotify.CloseWrite == fsnotify.CloseWrite {
				if event.Name == buddy.Lock.Path {
					err = buddy.Lock.Set()
					if err != nil {
						log.Println("lock error:", err)
					}
				} else if event.Name == buddy.ModePath {
					modeRaw, err := ioutil.ReadFile(buddy.ModePath)
					if err != nil {
						log.Println("mode error:", err)
						break
					}
					mode := string(modeRaw)
					mode = strings.TrimSpace(mode)
					buddy.SetBootMode(mode)
				} else {
					buddy.Enqueue(event.Name)
				}
			}
		case err := <-watcher.Errors:
			log.Println("fsnotify error:", err)
		case <-killChan:
			return errors.New("Terminated by signal.")
		}
	}
}

func (b *Buddy) InitLock() {
	b.Lock.set = false
	b.Lock.Path = filepath.Join(b.InDir, "lock")
	b.Lock.buddy = b
}

func main() {
	flag.Parse()
	if o.WatchDir[len(o.WatchDir)-1] != '/' {
		o.WatchDir += "/"
	}
	done := make(chan bool)
	for _, server := range Servers {
		box := o.ControlHosts[server]
		inDir := filepath.Join(o.WatchDir, box, "in")
		buddy := &Buddy{
			Servername: server,
			Boxname:    box,
			InDir:      inDir,
			queue:      make(chan o.WorkPackage),
			ModePath:   filepath.Join(inDir, "mode"),
		}
		buddy.InitLock()
		err := buddy.Connect()
		for err != nil {
			log.Println("RPC:", err)
			time.Sleep(time.Second)
			err = buddy.Connect()
		}

		go retryWatchAndRun(buddy, done)
		buddy.UpdateWebView(o.WebData{Mode: "batch"})
	}
	for range Servers {
		<-done
	}
}
