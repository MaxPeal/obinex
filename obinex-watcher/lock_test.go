// +build !integration

package main

import (
	"os"
	"testing"
	"time"
)

const PRE = "__obinex-test__"

func createLock(duration time.Duration) *Lock {
	lockpath := PRE + "testlockfile"
	dummyBuddy := &Buddy{
		InDir: "somerandomstringthatishopefullynotadirname",
	}
	lock := &Lock{
		uid:   uint32(os.Getuid()),
		Path:  lockpath,
		buddy: dummyBuddy,
	}

	f, _ := os.Create(lockpath)
	f.WriteString(time.Now().Add(time.Second).Format(time.RFC3339))
	f.Close()
	return lock
}

func createFile() string {
	path := PRE + "testfile"
	f, _ := os.Create(path)
	f.Close()
	return path
}

func TestGetUid(t *testing.T) {
	path := createFile()
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
	lock := createLock(time.Second)
	defer os.Remove(lock.Path)

	path := createFile()
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

func TestUnlock(t *testing.T) {
	lock := createLock(time.Minute)
	defer os.Remove(lock.Path)

	path := createFile()
	defer os.Remove(path)

	lock.Set()
	lock.uid = 0
	lock.Unset()
	time.Sleep(10 * time.Microsecond)
	ok := lock.Get(path)
	if !ok {
		t.Errorf("ok = false, want true\n")
	}
}
