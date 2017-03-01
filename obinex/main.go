package main

import (
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"path/filepath"
	"time"

	o "gitlab.cs.fau.de/luksen/obinex"
)

var (
	command  string
	box      string
	watchdir string
)

func init() {
	commands := `Valid commands:
	help: print this help
	lock: lock one of the boxes for yourself
`
	flag.StringVar(&command, "cmd", "help", "`command` to send to the server")
	flag.StringVar(&box, "box", o.CurrentBox(), "name of the hardwarebox you want to control")
	flag.StringVar(&watchdir, "watchdir", o.WatchDir, "`Path` to the directory being watched for binaries.")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, commands)
	}
}

func connect() *rpc.Client {
	host := o.HostByBox[box]
	client, err := rpc.DialHTTP("tcp", host+":12334")
	if err != nil {
		log.Println("dialing:", err)
		return nil
	}
	return client
}

func CmdLock(args []string) error {
	arg := ""
	for _, a := range args {
		arg += a
	}
	duration, err := time.ParseDuration(arg)
	if err != nil {
		return err
	}

	path := filepath.Join(watchdir, box, "in", "lock")
	log.Println(path)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	f.WriteString(time.Now().Add(duration).Format(time.RFC3339))
	f.Close()
	return nil
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	switch command {
	case "lock":
		err := CmdLock(flag.Args())
		if err != nil {
			log.Fatal(err)
		}
	default:
		flag.Usage()
	}
}
