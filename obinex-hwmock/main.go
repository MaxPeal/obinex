package main

import (
	"bufio"
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
		res, err := client.Get("http://localhost:12230/mock")
		for check(err) {
			log.Println("Waiting for server...")
			time.Sleep(1 * time.Second)
			res, err = client.Get("http://localhost:12230/mock")
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

		cmd := exec.Command("bash", "hwmock_tmp_script")
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Println(err)
			continue
		}

		err = cmd.Start()
		if err != nil {
			log.Println(err)
			continue
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Println(string(scanner.Bytes()))
			w.Write(scanner.Bytes())
			w.Write([]byte{byte('\n')})
		}
		if err := scanner.Err(); err != nil {
			log.Println(err)
			continue
		}

		exitCode := 0
		if err := cmd.Wait(); err != nil {
			log.Println(err)
			exitErr, ok := err.(*exec.ExitError)
			if ok {
				waitStatus := exitErr.Sys().(syscall.WaitStatus)
				exitCode = waitStatus.ExitStatus()
			} else {
				continue
			}
		}
		fmt.Fprintf(w, "octopos-shutdown %d\n", exitCode)
		os.Remove(f.Name())
		log.Println("Done")
	}
}

func main() {
	Run()
}
