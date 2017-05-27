package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"syscall"
	"time"

	o "gitlab.cs.fau.de/luksen/obinex"
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
	unlock  chan bool
	buddy   *Buddy //this is so we can trigger a walk when the lock expires
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
	l.unlock = make(chan bool, 1)
	go func() {
		c := time.After(l.Expires.Sub(time.Now()))
		select {
		case <-c:
			break
		case <-l.unlock:
			break
		}
		l.set = false
		os.Remove(l.Path)
		l.buddy.UpdateWebView(o.WebData{Lock: ""})
		log.Println("Lock: unlocked")
		l.buddy.walkAndRun(l.buddy.InDir, nil)
	}()

	l.buddy.UpdateWebView(o.WebData{Lock: "This machine is locked by " + o.Username(l.uid) + "."})
	log.Println("Lock: locked")
	return nil
}

func (l Lock) Unset() {
	select {
	case l.unlock <- true:
		break
	default:
		break
	}
}
