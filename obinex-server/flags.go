package main

import (
	"flag"

	o "gitlab.cs.fau.de/luksen/obinex"
)

var SerialPath string

func init() {
	flag.StringVar(&SerialPath, "serialpath", o.SerialPath, "`Path` to the serial node for talking to the hardware.")
	flag.StringVar(&o.WatchDir, "watchdir", o.WatchDir, "`Path` to the directory being watched for binaries.")
}
