// +build !integration

package main

import (
	"testing"

	o "github.com/maxpeal/obinex"
)

type MockLock struct {
	Block bool
}

func (l *MockLock) Init(buddy *Buddy)  {}
func (l MockLock) GetPath() string     { return "mocklocknopath" }
func (l MockLock) Get(bin string) bool { return !l.Block }
func (l *MockLock) Set() error         { return nil }
func (l MockLock) Unset()              {}
func (l MockLock) IsSet() bool         { return l.Block }
func (l MockLock) HolderUid() uint32   { return 0 }

func TestSetBootMode(t *testing.T) {
	oldExecCommand := o.ExecCommand
	defer func() { o.ExecCommand = oldExecCommand }()

	buddy := Buddy{
		Boxname: "testbox",
		Lock:    &MockLock{},
	}

	called := false
	o.ExecCommand = func(cmd string, args ...string) ([]byte, error) {
		called = true
		return nil, nil
	}

	buddy.SetBootMode("foo")
	if called {
		t.Error("ExecCommand called, should be invalid mode.")
	}

	buddy.Lock.(*MockLock).Block = true
	buddy.SetBootMode("linux")
	if called {
		t.Error("ExecCommand called, should be blocked by lock.")
	}

	o.ExecCommand = func(cmd string, args ...string) ([]byte, error) {
		if cmd == "bash" &&
			args[0] == "-c" &&
			args[1] == o.BootModePath+" testbox linux" {
			called = true
		}
		return nil, nil
	}
	buddy.Lock.(*MockLock).Block = false
	buddy.SetBootMode("linux")
	if !called {
		t.Error("ExecCommand called, expected no call.")
	}
}
