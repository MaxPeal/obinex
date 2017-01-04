package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"time"

	"github.com/kr/pty"
)

func check(err error) bool {
	if err, ok := err.(*url.Error); ok {
		if err, ok := err.Err.(*net.OpError); ok {
			if err, ok := err.Err.(*os.SyscallError); ok {
				if err.Err == syscall.ECONNREFUSED {
					return true
				}
			}
		}
	}
	if err != nil {
		log.Fatal(err)
	}
	return false
}

func Run() {
	w, tty, err := pty.Open()
	check(err)
	defer w.Close()
	defer tty.Close()
	log.Println("Providing serial device at", tty.Name())

	for {
		_, err := http.Get("http://localhost:12334/mock")
		for check(err) {
			log.Println("Waiting for server...")
			time.Sleep(1 * time.Second)
			_, err = http.Get("http://localhost:12334/mock")
		}
		log.Println("Giving output")
		io.WriteString(w, "executing\n")
		io.WriteString(w, "executing\n")
		io.WriteString(w, "executing\n")
		io.WriteString(w, "Graceful shutdown initiated\n")
	}
}

func main() {
	Run()
}
