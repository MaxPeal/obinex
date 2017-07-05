package main

import (
	"flag"

	o "gitlab.cs.fau.de/luksen/obinex"
)

func init() {
	flag.StringVar(&o.WatchDir, "watchdir", o.WatchDir, "`Path` to the directory being watched for binaries.")
	flag.Var(&o.Servers, "servers", "`List` of servers to connect to")
}
