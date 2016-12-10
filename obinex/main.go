package main

import (
	"flag"
	"fmt"
	"log"
	"net/rpc"
	"os"
)

var (
	command string
	host    string
)

func init() {
	commands := `Valid commands:
	help: print this help
	abandon: move on to the next binary
`
	flag.StringVar(&command, "cmd", "help", "`command` to send to the server")
	flag.StringVar(&host, "host", "localhost", "host that's running the server")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, commands)
	}
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	client, err := rpc.DialHTTP("tcp", host+":12334")
	if err != nil {
		log.Println("dialing:", err)
		return
	}
	defer client.Close()

	switch command {
	case "abandon":
		err = client.Call("Rpc.Abandon", struct{}{}, &struct{}{})
		if err != nil {
			log.Fatal("RPC Error:", err)
		}
	default:
		flag.Usage()
	}
}
