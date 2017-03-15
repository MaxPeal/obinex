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
	commands := `
Commands:
  help
    	print this help
  lock timestring
    	lock one of the boxes for yourself for the given duration

Timestring:
  A string that can be parsed as a duration, such as "30m" or "4h20m". The lock
  will be set to automatically expire after the given duration. Currently
  supported units are "h", "m" and "s".

Examples:
  To lock the fastbox for 24 hours you would run:

    	obinex -box fastbox -cmd lock 24h

File system interface:
  A lot of obinex actions (some of which are not supported by this tool) can be 
  executed through the file system at 'watchdir' (/proj/i4obinex/). See
  README.md or gitlab.cs.fau.de/luksen/obinex for documentation.
`
	flag.StringVar(&command, "cmd", "help", "`command` to execute")
	flag.StringVar(&box, "box", o.CurrentBox(), "name of the hardwarebox you want to control")
	flag.StringVar(&watchdir, "watchdir", o.WatchDir, "`path` to the directory being watched for binaries")
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
