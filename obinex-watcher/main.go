package main

import (
	"flag"
	"log"
	"net"
	"net/rpc"
	"os"
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

func handleOutput(box, path, s string) {
	t := time.Now().Format("_2006_01_02_15_04")
	// Create directories in out
	dir := strings.SplitN(path[len(WatchDir):], string(filepath.Separator), 3)[2]
	bin := filepath.Base(dir)
	dir = filepath.Dir(dir[:len(dir)-1])
	dir = filepath.Join(WatchDir, box, "out", dir, bin+t)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		log.Println("Output Error:", err)
		return
	}

	// Move bin
	err = os.Rename(path, filepath.Join(dir, bin))
	if err != nil {
		log.Println("Output Error:", err)
		return
	}

	// Write output file
	f, err := os.Create(filepath.Join(dir, "output.txt"))
	if err != nil {
		log.Println("Output Error:", err)
		return
	}
	defer f.Close()
	f.WriteString(s)
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

func retryWatchAndRun(name string, done chan bool) {
	defer func() { done <- true }()
	err := watchAndRun(name)
	for shouldRetry(err) {
		time.Sleep(1 * time.Second)
		err = watchAndRun(name)
	}
	log.Printf("watchAndRun [%s]: %s", name, err)
	return
}

func watchAndRun(name string) error {
	box := o.ControlHosts[name]

	client, err := rpc.DialHTTP("tcp", name+":12334")
	if err != nil {
		return err
	}
	defer client.Close()
	log.Printf("RPC: %s connected\n", name)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println(err)
		return err
	}
	defer watcher.Close()

	watching := WatchDir + "/" + box + "/in/"
	os.MkdirAll(watching, 0755)
	err = filepath.Walk(watching, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println(err)
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

	shutdown := make(chan error)
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				info, err := os.Stat(event.Name)
				if err != nil {
					log.Println(err)
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
				log.Println("Watcher: running", event.Name)
				go func() {
					s, err := run(client, event.Name)
					if err != nil {
						shutdown <- err
						return
					}
					handleOutput(box, event.Name, s)
				}()
			}
		case err := <-watcher.Errors:
			log.Println("fsnotify error:", err)
		case err := <-shutdown:
			return err
		}
	}
}

func main() {
	flag.Parse()
	done := make(chan bool)
	for _, server := range Servers {
		go retryWatchAndRun(server, done)
	}
	for range Servers {
		<-done
	}
}
