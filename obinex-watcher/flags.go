package main

import (
	"flag"
	"strings"

	o "gitlab.cs.fau.de/luksen/obinex"
)

type StringList []string

func (sl *StringList) String() string {
	return strings.Join(*sl, ",")
}

func (sl *StringList) Set(value string) error {
	*sl = StringList(strings.Split(value, ","))
	return nil
}

var Servers StringList

func init() {
	flag.StringVar(&o.WatchDir, "watchdir", o.WatchDir, "`Path` to the directory being watched for binaries.")
	Servers = StringList(o.Servers)
	flag.Var(&Servers, "servers", "`List` of servers to connect to")
}
