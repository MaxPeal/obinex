package main

import (
	"fmt"
	"log"
	"net/rpc"
)

func run(client *rpc.Client, bin string) string {
	var res string
	err := client.Call("Rpc.Run", bin, &res)
	if err != nil {
		log.Fatal("error:", err)
	}
	return res
}

func main() {
	client, err := rpc.DialHTTP("tcp", "localhost:12334")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	s := run(client, "../octopos-jenkins/testcase.microbench")
	fmt.Printf("%s", s)
}
