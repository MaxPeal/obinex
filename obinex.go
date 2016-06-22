package main

import (
	"fmt"
	"log"
	"net/rpc"
)

//import (
//"github.com/fsnotify/fsnotify"
//)

const watchDir = "/proj/i4invasic/obinex"

func run(client *rpc.Client, bin string) string {
	var res string
	err := client.Call("Rpc.Run", bin, &res)
	if err != nil {
		log.Fatal("rpc error:", err)
	}
	return res
}

/*
func watch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	err = watcher.Add(watchdir + "/fastbox")
	if err != nil {
		log.Fatal("fsnotify error:", err)
	}

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Create == fsnotify.Create {
				//add to list
				//q.PushBack(event.Name)
				//log.Println("----")
				//for e := q.Front(); e != nil; e = e.Next() {
				//log.Printf("%s\n", e.Value)
				//}
			}
		case err := <-watcher.Errors:
			log.Println("fsnotify error:", err)
		}
	}
}
*/

func main() {
	client, err := rpc.DialHTTP("tcp", "localhost:12334")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	s := run(client, "../octopos-jenkins/testcase.microbench")
	fmt.Printf("%s", s)
}
