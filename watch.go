package main

import "github.com/fsnotify/fsnotify"

import (
	"container/list"
	"log"
)

func main() {
	q := list.New()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Create == fsnotify.Create {
					//add to list
					q.PushBack(event.Name)
					log.Println("----")
					for e := q.Front(); e != nil; e = e.Next() {
						log.Printf("%s\n", e.Value)
					}
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add("/home/luki/invasic/binaryService")
	if err != nil {
		log.Fatal(err)
	}
	<-done
}
