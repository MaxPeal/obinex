package main

import (
	"log"
	"net/rpc"
	"os"
	"path/filepath"
	"time"
)

import (
	o "gitlab.cs.fau.de/luksen/obinex"

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
	box := o.ControlHosts[name]

	client, err := rpc.DialHTTP("tcp", name+":12334")
	if err != nil {
		log.Println("dialing:", err)
		return
	}
	defer client.Close()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println(err)
		return
	}
	defer watcher.Close()
	watching := watchDir + "/" + box
	err = watcher.Add(watching)
	if err != nil {
		log.Println("fsnotify error:", err)
		return
	}
	log.Println("Watcher: watching " + watching)

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Println("Watcher:", event.Name)
				s := run(client, event.Name)
				handleOutput(box, filepath.Base(event.Name), s)
			}
		case err := <-watcher.Errors:
			log.Println("fsnotify error:", err)
		}
	}
}

func main() {
	//hostname, err := os.Hostname()
	//if err != nil {
	//log.Fatal(err)
	//}
	// add other hosts here for 1 to N paradigm
	watchAndRun("faui49bello2")
	watchAndRun("faui49jenkins12")
	watchAndRun("faui49jenkins13")
	watchAndRun("faui49jenkins14")
}
