package main

import (
	"errors"
	"flag"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"path/filepath"
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

func (b *Buddy) Run(wp o.WorkPackage) error {
	err := b.rpc.Call("Rpc.Run", wp, nil)
	return err
}

func (b *Buddy) Close() {
	b.rpc.Close()
}

func (b *Buddy) Enqueue(path string) bool {
	if b.Lock.Get(path) {
		log.Println("Watcher: queueing", path)
		wp := o.WorkPackage{Path: path}
		err := wp.ToQueued()
		if err != nil {
			log.Println("Error:", err)
		}
		b.queue <- wp
		return true
	}
	log.Println("Watcher: blocked", path)
	return false
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
	err := watchAndRun(buddy)
	for shouldRetry(err) {
		log.Println("Connection error, retrying...")
		time.Sleep(1 * time.Second)
		err = watchAndRun(buddy)
	}
	log.Printf("watchAndRun [%s]: %s", buddy.Servername, err)
	return
}

func watchAndRun(buddy *Buddy) error {
	// Send the buddy.queued binaries to the server one after another
	// This function is currently located here because of the shutdown
	// channel.
	shutdown := make(chan error)
	go func(buddy *Buddy) {
		for wp := range buddy.queue {
			wp.ToExecuting()
			err := buddy.Run(wp)
			if err != nil {
				shutdown <- err
				return
			}
			wp.ToOut()
		}
	}(buddy)

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
					break
				}
				buddy.Enqueue(event.Name)
			}
		case err := <-watcher.Errors:
			log.Println("fsnotify error:", err)
		case err := <-shutdown:
			return err
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
		}
		buddy.InitLock()
		buddy.Connect()

		go retryWatchAndRun(buddy, done)
	}
	for range Servers {
		<-done
	}
}
