#!/bin/bash

murder() {
	sudo -u i4obinex ssh $1 killall $2
}

run() {
	sudo -u i4obinex ssh $1 "sh -c 'cd /proj/i4obinex/system/; nohup bin/$2 > /dev/null 2>$3.log &'"
}

murder i4jenkins obinex-watcher
murder faui49jenkins12 obinex-server
murder faui49jenkins13 obinex-server
murder faui49jenkins14 obinex-server
murder faui49jenkins15 obinex-server

run faui49jenkins15 "obinex-server" fastbox
run faui49jenkins14 "obinex-server" big03
run faui49jenkins13 "obinex-server" big02
run faui49jenkins12 "obinex-server" big01
sleep 2
run i4jenkins "obinex-watcher -servers faui49jenkins15,faui49jenkins14,faui49jenkins13,faui49jenkins12" watcher
