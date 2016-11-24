package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

import (
	o "gitlab.cs.fau.de/luksen/obinex"
)

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
	<-activateOutputChan
	binChan <- tmpfile.Name()
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
	<-activateOutputChan
	binChan <- "foo"
	<-done

	if c := w.Code; c != http.StatusInternalServerError {
		t.Errorf("code = %d, want %d", c, http.StatusInternalServerError)
	}
}

func TestHandleOutput(t *testing.T) {
	c := make(chan string)

	go handleOutput(c)
	defer func() { testDone <- true }()

	// test normal operation
	binQueue = []string{"foo"}
	activateOutputChan <- struct{}{}
	c <- "foo\n"
	c <- o.EndMarker + "\n"
	s := <-outputChan
	if s != "foo\n"+o.EndMarker+"\n" {
		t.Errorf("string = %s, want fooGraceful...", s)
	}

	// test abandoned bin
	binQueue = []string{} // queue is too short
	activateOutputChan <- struct{}{}
	c <- o.EndMarker + "\n"
	s = <-outputChan
	if s != o.EndMarker+"\n" {
		t.Errorf("string = %s, want %s", s, o.EndMarker)
	}

	// test late detection
	binQueue = []string{"foo"}
	activateOutputChan <- struct{}{}
	c <- "foo\n"
	activateOutputChan <- struct{}{}
	s = <-outputChan
	if s != "foo\n" {
		t.Errorf("string = %s, want foo", s)
	}

	c <- o.EndMarker + "\n"
	<-outputChan
}
}
