package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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

	tr := &http.Transport{
		DisableKeepAlives: true,
	}
	client := &http.Client{Transport: tr}

	for {
		res, err := client.Get("http://localhost:12334/mock")
		for check(err) {
			log.Println("Waiting for server...")
			time.Sleep(1 * time.Second)
			res, err = client.Get("http://localhost:12334/mock")
		}
		log.Println("Executing")
		f, err := os.Create("hwmock_tmp_script")
		if err != nil {
			res.Body.Close()
			io.WriteString(w, fmt.Sprintln(err))
			continue
		}
		io.Copy(f, res.Body)
		res.Body.Close()
		f.Close()
		out, _ := exec.Command("bash", "hwmock_tmp_script").CombinedOutput()
		w.Write(out)
		os.Remove(f.Name())
		log.Println("Done")
	}
}

func main() {
	Run()
}
