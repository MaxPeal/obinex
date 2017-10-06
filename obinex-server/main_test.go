// +build !integration

package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/kr/pty"

	o "gitlab.cs.fau.de/i4/obinex"
	"golang.org/x/net/websocket"
)

const emptyWebDataString = "{\"LogLine\":\"\",\"Queue\":[],\"Lock\":\"\",\"Mode\":\"\"}"

func TestBinaryServeHandler(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	tmpfile, err := ioutil.TempFile("", "tmptestfile")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte("foo")); err != nil {
		log.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		log.Fatal(err)
	}

	// test normal operation
	done := make(chan bool)
	go func() {
		binaryServeHandler(w, r)
		done <- true
	}()
	<-lateEoeChan
	runToServChan <- o.WorkPackage{Path: tmpfile.Name()}
	<-servToOutChan
	<-done

	if b := w.Body.String(); !strings.Contains(b, "foo") {
		t.Errorf("body = %s, want foo", b)
	}

	// test without file
	w = httptest.NewRecorder()
	go func() {
		binaryServeHandler(w, r)
		done <- true
	}()
	<-lateEoeChan
	runToServChan <- o.WorkPackage{Path: "foo"}
	<-done

	if c := w.Code; c != http.StatusInternalServerError {
		t.Errorf("code = %d, want %d", c, http.StatusInternalServerError)
	}
}

func TestGetSerialOutput(t *testing.T) {
	w, tty, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	defer tty.Close()
	SerialPath = tty.Name()

	c := make(chan string)
	go getSerialOutput(c)

	io.WriteString(w, "foobar\n")
	if s := <-c; s != "foobar\n" {
		t.Errorf("channel = %s, want foobar", s)
	}
	testDone <- true
	io.WriteString(w, "hanging read\n")
	<-c
}

func TestHandleOutput(t *testing.T) {
	c := make(chan string)

	go handleOutput(c)
	defer func() { testDone <- true }()
	defer os.Remove("output.txt")

	outputChan := make(chan o.WebData)
	wsAddChan <- outputChan

	// test normal operation
	persistentWebData.Queue = []string{"foo"}
	servToOutChan <- o.WorkPackage{Path: "foo"}
	c <- "foo\n"
	c <- o.EndMarker + "0\n"
	<-eoeChan
	if len(persistentWebData.Queue) != 0 {
		t.Errorf("len(persistentWebData.Queue) = %d, expected 0", len(persistentWebData.Queue))
	}
	f, _ := os.Open("output.txt")
	b, _ := ioutil.ReadAll(f)
	f.Close()
	s := string(b)
	if s != "foo\n"+o.EndMarker+"0\n" {
		t.Errorf("s = %s, expected \"foo\\n%s0\\n", s, o.EndMarker)
	}

	// test late detection
	persistentWebData.Queue = []string{"foo"}
	servToOutChan <- o.WorkPackage{Path: "foo"}
	c <- "foo\n"
	lateEoeChan <- struct{}{}
	<-eoeChan
	if len(persistentWebData.Queue) != 0 {
		t.Errorf("len(persistentWebData.Queue) = %d, expected 0", len(persistentWebData.Queue))
	}
	f, _ = os.Open("output.txt")
	b, _ = ioutil.ReadAll(f)
	f.Close()
	s = string(b)
	if s != "foo\n" {
		t.Errorf("s = %s, expected \"foo\\n", s)
	}
}

func TestRun(t *testing.T) {
	rpc := Rpc{}
	in := o.WatchDir + "mock/executing/somedir/somebin"
	done := make(chan bool)
	err := error(nil)

	go func() { err = rpc.Run(o.WorkPackage{Path: in}, nil); done <- true }()
	defer func() { persistentWebData.Queue = []string{} }()
	wp := <-runToServChan
	eoeChan <- struct{}{}
	<-done

	if wp.Path != o.WatchDir+"mock/executing/somedir/somebin" {
		t.Errorf("bin = %s, want somebin", o.WatchDir+"mock/executing/somedir/somebin")
	}
	if err != nil {
		t.Errorf("error = %s, want nil", err)
	}
	if persistentWebData.Queue[0] != "somedir/somebin" {
		t.Errorf("persistentWebData.Queue = %v, want somedir/somebin", persistentWebData.Queue)
	}
}

func TestWebsocket(t *testing.T) {
	server := httptest.NewServer(websocket.Handler(websocketHandler))
	defer server.Close()

	conn1, err := websocket.Dial("ws://"+server.URL[7:], "", "http://localhost")
	if err != nil {
		t.Fatal(err)
	}
	defer conn1.Close()
	conn2, err := websocket.Dial("ws://"+server.URL[7:], "", "http://localhost")
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close()

	wsChan <- o.WebData{}

	var buf = make([]byte, 512)
	var n int
	if n, err = conn1.Read(buf); err != nil {
		log.Fatal(err)
	}
	if string(buf[:n]) != emptyWebDataString {
		t.Errorf("output = %s, want empty o.WebData", string(buf[:n]))
	}

	if n, err = conn2.Read(buf); err != nil {
		log.Fatal(err)
	}
	if string(buf[:n]) != emptyWebDataString {
		t.Errorf("output = %s, want empty o.WebData", string(buf[:n]))
	}
}
