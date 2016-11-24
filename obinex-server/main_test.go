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
		t.Error("body = %s, want foo", b)
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
		t.Error("code = %d, want %d", c, http.StatusInternalServerError)
	}
}
