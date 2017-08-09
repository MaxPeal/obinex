package main

import (
	"flag"

	o "gitlab.cs.fau.de/luksen/obinex"
)

var SerialPath string
var Boxname string

// Boxes lists the hardware boxes served by default
var Boxes o.StringList

func init() {
	flag.StringVar(&SerialPath, "serialpath", o.SerialPath, "`Path` to the serial node for talking to the hardware.")
	flag.StringVar(&Boxname, "box", "mock", "Name of the hardware box corresponding to this server instance.")
	flag.StringVar(&o.WatchDir, "watchdir", o.WatchDir, "`Path` to the directory being watched for binaries.")
	flag.Var(&Boxes, "boxes", "`List` of hardware boxes being served")
}
