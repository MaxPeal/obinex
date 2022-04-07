package main

import (
	"flag"

	o "github.com/maxpeal/obinex"
)

var Boxname string

func init() {
	flag.StringVar(&o.ConfigPath, "config", o.ConfigPath, "`Path` to the configuration file.")
	flag.StringVar(&Boxname, "box", "mock", "Name of the hardware box corresponding to this server instance.")
	flag.StringVar(&o.WatchDir, "watchdir", o.WatchDir, "`Path` to the directory being watched for binaries.")
	flag.StringVar(&o.SerialPath, "serialpath", o.SerialPath, "`Path` to the serial node for talking to the hardware.")
	flag.Var(&o.Boxes, "boxes", "`List` of hardware boxes being served")
}
