package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

// Server stuff
type Args struct {
	A, B int
}

type Res struct {
	X int
}

type Test int

func (t *Test) Add(args *Args, reply *Res) error {
	reply.X = args.A + args.B
	return nil
}

func main() {
	// server
	test := new(Test)
	rpc.Register(test)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":1234")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)

	//clinet
	client, err := rpc.DialHTTP("tcp", "localhost:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	//sync
	args := &Args{3, 4}
	var res Res
	err = client.Call("Test.Add", args, &res)
	if err != nil {
		log.Fatal("error:", err)
	}
	fmt.Printf("%d\n", res.X)

	//async
	args = &Args{5, 6}
	call := client.Go("Test.Add", args, &res, nil)
	rep := <-call.Done
	if rep.Error != nil {
		log.Fatal("error:", rep.Error)
	}
	fmt.Printf("%d\n", (rep.Reply.(*Res)).X)
}
