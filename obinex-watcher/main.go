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
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	o "gitlab.cs.fau.de/luksen/obinex"
)

func run(client *rpc.Client, bin string) (string, error) {
	var res string
	err := client.Call("Rpc.Run", bin, &res)
	return res, err
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

func toQueued(bin string) string {
	org := bin
	// Create new structure
	t := time.Now().Format("2006-01-02_-_15:04:05")
	dir := filepath.Dir(bin) + "/"
	bin = filepath.Base(bin)
	dir = changeStateOnPath(dir, "queued")
	dir = filepath.Join(dir, bin+"_"+t)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		log.Println("Output Error:", err)
		return ""
	}

	// Move bin
	err = os.Rename(org, filepath.Join(dir, bin))
	if err != nil {
		log.Println("Output Error:", err)
	}
	return filepath.Join(dir, bin)
}

func toY(bin, y string) string {
	dir := filepath.Dir(bin) + "/"
	new := changeStateOnPath(dir, y)
	err := os.MkdirAll(filepath.Join(new, ".."), 0755)
	if err != nil {
		log.Println("Output Error:", err)
		return ""
	}

	// Move dir
	err = os.Rename(dir, new)
	if err != nil {
		log.Println("Output Error:", err)
	}
	return filepath.Join(new, filepath.Base(bin))
}

func toExecuting(bin string) string {
	return toY(bin, "executing")
}

func toOut(bin string) {
	toY(bin, "out")
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

func retryWatchAndRun(name string, queue chan string, done chan bool) {
	defer func() { done <- true }()
	err := watchAndRun(name, queue)
	for shouldRetry(err) {
		time.Sleep(1 * time.Second)
		err = watchAndRun(name, queue)
	}
	log.Printf("watchAndRun [%s]: %s", name, err)
	return
}

func watchAndRun(name string, queue chan string) error {
	box := o.ControlHosts[name]

	log.Println("RPC: connecting to", name)
	client, err := rpc.DialHTTP("tcp", name+":12334")
	if err != nil {
		return err
	}
	defer client.Close()
	log.Printf("RPC: %s connected\n", name)

	// Send the queued binaries to the server one after another
	shutdown := make(chan error)
	go func(client *rpc.Client, queue chan string) {
		for bin := range queue {
			bin = toExecuting(bin)
			log.Println(bin)
			output, err := run(client, bin)
			if err != nil {
				shutdown <- err
				return
			}
			// Write output file
			f, err := os.Create(filepath.Join(filepath.Dir(bin), "output.txt"))
			if err != nil {
				shutdown <- err
				return
			}
			f.WriteString(output)
			f.Close()
			toOut(bin)
		}
	}(client, queue)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("Watcher:", err)
		return err
	}
	defer watcher.Close()

	watching := filepath.Join(WatchDir, box, "in")
	os.MkdirAll(watching, 0755)
	err = filepath.Walk(watching, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println("Watcher:", err)
			return err
		}
		if info.IsDir() == false {
			log.Printf("Watcher: not a directory: %s\n", path)
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

	killChan := make(chan os.Signal, 1)
	signal.Notify(killChan, syscall.SIGTERM)
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(event.Name)
				if err != nil {
					log.Println("Watcher:", err)
					break
				}
				if info.IsDir() {
					err = watcher.Add(event.Name)
					if err != nil {
						log.Println("fsnotify error:", err)
					}
					log.Println("Watcher: watching " + event.Name)
					break
				}
				log.Println("Watcher: queueing", event.Name)
				path := toQueued(event.Name)
				queue <- path
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

func main() {
	flag.Parse()
	if WatchDir[len(WatchDir)-1] != '/' {
		WatchDir += "/"
	}
	done := make(chan bool)
	for _, server := range Servers {
		queue := make(chan string)
		go retryWatchAndRun(server, queue, done)
	}
	for range Servers {
		<-done
	}
}
