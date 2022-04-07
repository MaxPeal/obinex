package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	o "github.com/maxpeal/obinex"
)

type Locker interface {
	Get(string) bool
	Set() error
	Unset()
	IsSet() bool
	GetPath() string
	Init(*Buddy)
	HolderUid() uint32
}

func (l *Lock) Init(buddy *Buddy) {
	l.set = false
	l.Path = filepath.Join(buddy.InDir, "lock")
	l.buddy = buddy
}

func (l Lock) GetPath() string {
	return l.Path
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
	uid, err := o.Getuid(bin)
	if err != nil {
		log.Println("Lock:", err)
		return false
	}
	return l.uid == uid
}

func (l *Lock) Set() error {
	uid, err := o.Getuid(l.Path)
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

func (l Lock) IsSet() bool {
	return l.set
}

// HolderUid returns the uid of the lock holder. Only valid if IsSet() == true.
func (l Lock) HolderUid() uint32 {
	return l.uid
}
