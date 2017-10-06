package main

import (
	"flag"

	o "gitlab.cs.fau.de/i4/obinex"
)

var Host string

// Boxes lists the hardware boxes served by default
var Boxes = o.StringList{
	"faui49big01",
	"faui49big02",
	"faui49big03",
	"fastbox",
}

func init() {
	flag.StringVar(&Host, "host", "localhost", "Hostname of the machine running obinex-server instances.")
	flag.StringVar(&o.WatchDir, "watchdir", o.WatchDir, "`Path` to the directory being watched for binaries.")
	flag.Var(&Boxes, "boxes", "`List` of hardware boxes being served")
}
