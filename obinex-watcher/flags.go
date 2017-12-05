package main

import (
	"flag"

	o "gitlab.cs.fau.de/i4/obinex"
)

var Host string

func init() {
	flag.StringVar(&o.ConfigPath, "config", o.ConfigPath, "`Path` to the configuration file.")
	flag.StringVar(&Host, "host", "localhost", "Hostname of the machine running obinex-server instances.")
	flag.StringVar(&o.WatchDir, "watchdir", o.WatchDir, "`Path` to the directory being watched for binaries.")
	flag.Var(&o.Boxes, "boxes", "`List` of hardware boxes being served")
}
