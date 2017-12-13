#!/bin/bash

murder() {
	sudo -u i4obinex ssh -o StrictHostKeyChecking=no $1 killall $2
}

run() {
	sudo -u i4obinex ssh -o StrictHostKeyChecking=no $1 "sh -c 'cd /proj/i4obinex/system/; nohup bin/$2 > /dev/null 2>>$3.log &'"
}

murder i4jenkins obinex-watcher
murder faui49obinex obinex-server
sleep 5

run faui49obinex "obinex-server -box fastbox" fastbox
run faui49obinex "obinex-server -box faui49big03" big03
run faui49obinex "obinex-server -box faui49big02" big02
run faui49obinex "obinex-server -box faui49big01" big01
sleep 2
run i4jenkins "obinex-watcher -host faui49obinex" watcher
