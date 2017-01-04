package main

import (
	"flag"

	o "gitlab.cs.fau.de/luksen/obinex"
)

var WatchDir string

func init() {
	flag.StringVar(&WatchDir, "watchdir", o.WatchDir, "`Path` to the directory being watched for binaries.")
}
