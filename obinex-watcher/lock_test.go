// +build !integration

package main

import (
	"os"
	"testing"
	"time"
)

const PRE = "__obinex-test__"

func TestGetUid(t *testing.T) {
	path := PRE + "testfile"
	f, _ := os.Create(path)
	f.Close()
	defer os.Remove(path)

	uid := os.Getuid()
	u, err := getUid(path)
	if err != nil {
		t.Error(err)
	}
	if int(u) != uid {
		t.Errorf("uid = %d, want %d\n", u, uid)
	}
}

func TestLock(t *testing.T) {
	lockpath := PRE + "testlockfile"
	lock := &Lock{
		uid:  uint32(os.Getuid()),
		Path: lockpath,
	}

	f, _ := os.Create(lockpath)
	f.WriteString(time.Now().Add(time.Second).Format(time.RFC3339))
	f.Close()
	defer os.Remove(lockpath)

	path := PRE + "testfile"
	f, _ = os.Create(path)
	f.Close()
	defer os.Remove(path)

	lock.Set()
	ok := lock.Get(path)
	if !ok {
		t.Errorf("ok = false, want true\n")
	}

	lock.uid = 0
	ok = lock.Get(path)
	if ok {
		t.Errorf("ok = true, want false\n")
	}
}
