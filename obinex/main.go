package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"github.com/kardianos/osext"

	o "gitlab.cs.fau.de/i4/obinex"
)

var (
	command string
	box     string
	userdir string
)

func init() {
	userdir = "."
	user, err := user.Current()
	if err != nil {
		return
	}
	userdir = user.Username
}

func init() {
	commands := `
Commands:
  help
    	print this help
  lock [timestring]
    	lock one of the boxes for yourself for the given duration or give information about the lock
  run <binary|ipxe script> [parameters]
    	submit the binary or ipxe script for execution (boot parameters are optional)
  output <binary>
    	get output for the most recently submitted binary with this name
  reset
    	reset the hardware box, causing it to reboot
  mode [bootmode]
    	set the boot mode for the hardware box, if there are no binaries queued you might want to run 'reset' afterwards

Timestring:
  A string that can be parsed as a duration, such as "30m" or "4h20m". The lock
  will be set to automatically expire after the given duration. Currently
  supported units are "h", "m" and "s".

Bootmode:
  After changing the mode you can either wait for the currently running binary
  to finish (and cause a reboot) or manually run 'reset'. Valid modes are:
    - linux: boot Linux on the hardware box
    - batch: run binaries from obinex (default and normal operation)

Examples:
  To lock the fastbox for 24 hours, you would run:

    	obinex -box fastbox -cmd lock 24h

  To execute a binary, run:

    	obinex -box faui49big01 -cmd run mybin

  To execute a binary with boot parameters, run:

    	obinex -box faui49big01 -cmd run mybin param1 param2...

  To get the output from your last submitted binary, run:

    	obinex -box faui49big01 -cmd output mybin

File system interface:
  A lot of obinex actions can be executed through the file system at 'o.WatchDir'
  (/proj/i4obinex/). See README.md or gitlab.cs.fau.de/i4/obinex for
  documentation.
`
	flag.StringVar(&command, "cmd", "help", "`command` to execute")
	flag.StringVar(&box, "box", "mock", "name of the hardwarebox you want to control")
	flag.StringVar(&o.WatchDir, "watchdir", o.WatchDir, "`path` to the directory being watched for binaries")
	flag.StringVar(&userdir, "userdir", userdir, "name of your personal subdirectory")
	flag.StringVar(&o.WatcherHost, "watcherhost", o.WatcherHost, "host where obinex-watcher is running")
	flag.StringVar(&o.ConfigPath, "config", o.ConfigPath, "`Path` to the configuration file.")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, commands)
	}
}

func closeErr(c io.Closer) {
	err := c.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer closeErr(in)
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	close_err := out.Close()
	if err != nil {
		return err
	}
	return close_err
}

type CommandFunction func([]string) error

var Commands map[string]CommandFunction = map[string]CommandFunction{
	"help":   CmdHelp,
	"lock":   CmdLock,
	"run":    CmdRun,
	"output": CmdOutput,
	"reset":  CmdReset,
	"mode":   CmdMode,
}

func CmdHelp(args []string) error {
	flag.Usage()
	return nil
}

func CmdLock(args []string) error {
	arg := strings.Join(args, "")
	path := filepath.Join(o.WatchDir, box, "in", "lock")

	if arg == "" {
		content, err := ioutil.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("No lock")
				return nil
			}
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
		duration := date.Sub(time.Now())
		log.Printf("Lock expires in %v", duration)
		return nil
	}

	duration, err := time.ParseDuration(arg)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	f.WriteString(time.Now().Add(duration).Format(time.RFC3339))
	f.Close()
	log.Printf("Locked %s for %v", box, duration)
	return nil
}

func CmdRun(args []string) error {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	bin := args[0]
	parameters := ""
	if len(args) > 1 {
		parameters = strings.Join(args[1:], " ")
	}
	target := filepath.Join(o.WatchDir, box, "in", userdir, filepath.Base(bin)+"_"+id)

	err := os.MkdirAll(filepath.Dir(target), 0775)
	if err != nil {
		log.Println(err)
	}
	// The chmod is needed because go uses the  mkdirat syscall for
	// os.MkdirAll which leads to the group bit not being propagated
	// through NFS properly (not sure why). Explicitly setting the mode
	// again fixes this.
	err = os.Chmod(filepath.Dir(target), 0775)
	if err != nil {
		log.Println(err)
	}

	client, err := rpc.DialHTTP("tcp", o.WatcherHost+":12344")
	if err != nil {
		return err
	}
	arg := o.RpcArg{
		Boxname:    box,
		FileId:     id,
		Parameters: parameters,
	}
	err = client.Call("Rpc.RunWithParameters", arg, &struct{}{})
	if err != nil {
		return err
	}

	return copyFile(bin, target)
}

func CmdOutput(args []string) error {
	name := strings.Join(args, " ")
	boxdir := filepath.Join(o.WatchDir, box)

	var mostRecentDate time.Time
	var mostRecentDir string
	var mostRecentStatus string
	for _, dir := range []string{"queued", "executing", "out"} {
		prefix := filepath.Join(boxdir, dir, userdir, name) + "_"
		dateDirs, err := filepath.Glob(prefix + "*")
		if err != nil {
			return err
		}
		for _, dd := range dateDirs {
			date, _ := time.Parse(o.DirectoryDateFormat, dd[len(prefix):])
			if date.After(mostRecentDate) {
				mostRecentDate = date
				mostRecentDir = dd
				mostRecentStatus = dir
			}
		}
	}

	switch mostRecentStatus {
	case "out":
		outFile, err := os.Open(filepath.Join(mostRecentDir, "output.txt"))
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		defer outFile.Close()

		_, err = io.Copy(os.Stdout, outFile)
		if err != nil {
			return err
		}

	case "executing":
		outFile, err := os.Open(filepath.Join(mostRecentDir, "output.txt"))
		if err != nil {
			return err
		}
		defer outFile.Close()

		_, err = io.Copy(os.Stdout, outFile)
		if err != nil {
			return err
		}

	case "queued":
		log.Println("Your binary is queued.")
	}
	return nil
}

func CmdReset(args []string) error {
	client, err := rpc.DialHTTP("tcp", o.WatcherHost+":12344")
	if err != nil {
		return err
	}
	uid := uint32(syscall.Getuid())
	var output string
	arg := o.RpcArg{
		Boxname: box,
		Uid:     uid,
	}
	err = client.Call("Rpc.Reset", arg, &output)
	log.Println(output)
	return err
}

func CmdMode(args []string) error {
	arg := strings.Join(args, " ")
	if arg != "linux" &&
		arg != "batch" &&
		arg != "nfs" &&
		arg != "interactive" {
		return errors.New("Invalid mode. Use linux, batch, nfs or interactive.")
	}
	path := filepath.Join(o.WatchDir, box, "in", "mode")
	os.Remove(path)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = f.WriteString(arg)
	return err
}

func main() {
	log.SetFlags(0)

	// If a configuration file exists in the current working directory, use it;
	// otherwise search for it in the binary's parent directory.
	configPath := o.ConfigPath
	_, err := os.Stat(configPath)
	if err != nil {
		exeDir, err := osext.ExecutableFolder()
		if err != nil {
			panic(err)
		}
		configPath = filepath.Join(filepath.Dir(exeDir), o.ConfigPath)
	}
	o.ReadConfig(configPath, "")

	flag.Parse()

	function, ok := Commands[command]
	if !ok {
		log.Fatalf("Unknown command `%s`, see command `help` for usage.", command)
	}
	err = function(flag.Args())
	if err != nil {
		log.Fatal(err)
	}
}
