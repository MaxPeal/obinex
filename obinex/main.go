package main

import (
	"log"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
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

func handleOutput(box, path, s string) {
	t := time.Now().Format("_2006_01_02_15_04")
	// Create directories in out
	dir := strings.SplitN(path, string(filepath.Separator), 7)[6]
	bin := filepath.Base(dir)
	dir = filepath.Dir(dir[:len(dir)-1])
	dir = filepath.Join(watchDir, box, "out", dir, bin+t)
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

	watching := watchDir + "/" + box + "/in/"
	err = filepath.Walk(watching, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() == false {
			return nil
		}

		err = watcher.Add(path)
		if err != nil {
			log.Println("fsnotify error:", err)
			return nil
		}
		log.Println("Watcher: watching " + path)
		return nil
	})

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
					s := run(client, event.Name)
					handleOutput(box, event.Name, s)
				}()
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
	watchAndRun("faui49jenkins12")
	watchAndRun("faui49jenkins13")
	watchAndRun("faui49jenkins14")
	watchAndRun("faui49jenkins15")
}
