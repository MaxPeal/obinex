package main

import (
	"log"
	"net/rpc"
	"os"
	"path/filepath"
	"time"
)

import (
	"github.com/fsnotify/fsnotify"
)

const watchDir = "/proj/i4invasic/obinex/"

func run(client *rpc.Client, bin string) string {
	var res string
	err := client.Call("Rpc.Run", bin, &res)
	if err != nil {
		log.Fatal("RPC Error:", err)
	}
	return res
}

func handleOutput(box, bin, s string) {
	t := time.Now().Format("_2006_01_02_15_04")
	f, err := os.Create(watchDir + "/" + box + "/out/" + bin + t + ".txt")
	if err != nil {
		log.Println("Output Error:", err)
		return
	}
	defer f.Close()
	f.WriteString(s)
}

func watchAndRun(name string) {
	client, err := rpc.DialHTTP("tcp", "localhost:12334")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer client.Close()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	err = watcher.Add(watchDir + "/" + name)
	if err != nil {
		log.Fatal("fsnotify error:", err)
	}
	log.Println("Watcher: watching " + watchDir + "/" + name)

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Println("Watcher:", event.Name)
				s := run(client, event.Name)
				handleOutput(name, filepath.Base(event.Name), s)
			}
		case err := <-watcher.Errors:
			log.Println("fsnotify error:", err)
		}
	}
}

func main() {
	watchAndRun("fastbox")
}
