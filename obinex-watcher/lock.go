package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"syscall"
	"time"
)

func getUid(path string) (uint32, error) {
	info, _ := os.Stat(path)
	val, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, errors.New("No Stat_t type")
	}
	return (*val).Uid, nil
}

type Lock struct {
	Expires time.Time
	uid     uint32
	set     bool
	Path    string
}

func (l Lock) Get(bin string) bool {
	if !l.set {
		return true
	}
	uid, err := getUid(bin)
	if err != nil {
		log.Println("Lock:", err)
		return false
	}
	return l.uid == uid
}

func (l *Lock) Set() error {
	uid, err := getUid(l.Path)
	if err != nil {
		return err
	}
	content, err := ioutil.ReadFile(l.Path)
	if err != nil {
		return err
	}
	datestring := strings.TrimSpace(string(content))
	date, err := time.Parse(time.RFC3339, datestring)
	if err != nil {
		format := strings.Replace(time.RFC3339, "T", " ", 1)
		date, err = time.Parse(format, datestring)
		if err != nil {
			return err
		}
	}
	l.uid = uid
	l.Expires = date
	l.set = true
	go func() {
		c := time.After(time.Until(l.Expires))
		<-c
		l.set = false
		os.Remove(l.Path)
		log.Println("Lock: unlocked")
	}()
	log.Println("Lock: locked")
	return nil
}
