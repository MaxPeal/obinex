package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	o "gitlab.cs.fau.de/i4/obinex"
)

type Buddy struct {
	Boxname        string
	Servername     string
	InDir          string
	ModePath       string
	ResetPath      string
	Lock           Locker
	queue          chan o.WorkPackage
	rpc            *rpc.Client
	parameterIndex map[string]string
}

type Rpc []*Buddy

func (r Rpc) Reset(arg o.RpcArg, output *string) error {
	log.Printf("RPC: reset %s\n", arg.Boxname)
	for _, b := range r {
		if b.Boxname == arg.Boxname {
			if b.Lock.IsSet() {
				if b.Lock.HolderUid() != arg.Uid {
					*output = "Locked by " + o.Username(b.Lock.HolderUid())
					return nil
				}
			}

			err := b.rpc.Call("Rpc.Powercycle", struct{}{}, &output)
			if err != nil {
				log.Println("RPC:", err)
			}
			return err
		}
	}
	log.Printf("RPC: invalid boxname %s\n", arg.Boxname)
	return fmt.Errorf("RPC: invalid boxname %s\n", arg.Boxname)
}

func (r Rpc) RunWithParameters(arg o.RpcArg, _ *struct{}) error {
	log.Printf("RPC: register parameters %s on %s\n", arg.FileId+" "+arg.Parameters, arg.Boxname)
	for _, b := range r {
		if b.Boxname == arg.Boxname {
			b.parameterIndex[arg.FileId] = arg.Parameters
			return nil
		}
	}
	log.Printf("RPC: invalid boxname %s\n", arg.Boxname)
	return fmt.Errorf("RPC: invalid boxname %s\n", arg.Boxname)
}

func NewBuddy(box string) (buddy *Buddy) {
	inDir := filepath.Join(o.WatchDir, box, "in")
	buddy = &Buddy{
		Servername:     Host,
		Boxname:        box,
		InDir:          inDir,
		ModePath:       filepath.Join(inDir, "mode"),
		ResetPath:      filepath.Join(inDir, "reset"),
		Lock:           &Lock{},
		queue:          make(chan o.WorkPackage),
		parameterIndex: make(map[string]string),
	}
	buddy.Lock.Init(buddy)
	return
}

func (b *Buddy) Connect() error {
	log.Println("RPC: connecting to", b.Servername)
	client, err := rpc.DialHTTP("tcp", b.Servername+o.PortByBox[b.Boxname])
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
		for id, params := range b.parameterIndex {
			if strings.HasSuffix(path, "_"+id) {
				wp.Parameters = params
				wp.FromCLT = true
				delete(b.parameterIndex, id)
			}
		}
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
		mode != "batch" {
		log.Printf("Invalid mode \"%s\". Mode not changed.\n", mode)
		return
	}
	log.Printf("Changing %s mode to \"%s\"\n", b.Boxname, mode)

	if !b.Lock.Get(b.ModePath) {
		log.Println("Mode change denied by lock. If you are the owner of the lock, try removing the mode file first.")
		return
	}

	_, err := o.ExecCommand("bash", "-c", o.BootModePath+" "+b.Boxname+" "+mode)
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
		if path == b.Lock.GetPath() {
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
				if event.Name == buddy.Lock.GetPath() {
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
				if event.Name == buddy.Lock.GetPath() {
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
					if _, err := os.Stat(event.Name); err == nil {
						// path does exist
						buddy.Enqueue(event.Name)
					}
				}
			}
		case err := <-watcher.Errors:
			log.Println("fsnotify error:", err)
		case <-killChan:
			return errors.New("Terminated by signal.")
		}
	}
}

func main() {
	o.ReadConfig(o.ConfigPath, "")
	flag.Parse()
	if o.WatchDir[len(o.WatchDir)-1] != '/' {
		o.WatchDir += "/"
	}

	buddyRpc := Rpc{}
	done := make(chan bool)
	for _, box := range o.Boxes {
		buddy := NewBuddy(box)
		err := buddy.Connect()
		if err == nil {
			buddyRpc = append(buddyRpc, buddy)
			go retryWatchAndRun(buddy, done)
		} else {
			log.Println("RPC:", err)
		}
	}

	rpc.Register(&buddyRpc)
	rpc.HandleHTTP()
	server := &http.Server{
		Addr: ":12344",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			addr := req.RemoteAddr
			// only allow access from these addresses
			if strings.HasPrefix(addr, "131.188.42.") ||
				strings.HasPrefix(addr, "10.188.42.") ||
				strings.HasPrefix(addr, "[2001:638:a000:4142:") ||
				strings.HasPrefix(addr, "[::1]") ||
				strings.HasPrefix(addr, "127.0.0.1") {

				http.DefaultServeMux.ServeHTTP(w, req)
				return
			}
			http.Error(w, "", 404)
			log.Printf("Blocked access from %s\n", addr)
		}),
	}
	go server.ListenAndServe()

	for range o.Boxes {
		<-done
	}
}
