#!/bin/bash

boxes="faui49big01,faui49big02,faui49big03,fastbox"

murder() {
	sudo -u i4obinex ssh $1 killall $2
}

run() {
	sudo -u i4obinex ssh $1 "sh -c 'cd /proj/i4obinex/system/; nohup bin/$2 -boxes $boxes > /dev/null 2>$3.log &'"
}

murder i4jenkins obinex-watcher
murder faui49jenkins25 obinex-server
murder faui49jenkins25 obinex-server
murder faui49jenkins25 obinex-server
murder faui49jenkins25 obinex-server

run faui49jenkins25 "obinex-server -box fastbox -serialpath /dev/ttyS7" fastbox
run faui49jenkins25 "obinex-server -box faui49big03 -serialpath /dev/ttyS6" big03
run faui49jenkins25 "obinex-server -box faui49big02 -serialpath /dev/ttyS5" big02
run faui49jenkins25 "obinex-server -box faui49big01 -serialpath /dev/ttyS4" big01
sleep 2
run i4jenkins "obinex-watcher -host faui49jenkins25" watcher
