#!/bin/bash


function global_setup() {
	go build gitlab.cs.fau.de/luksen/obinex/obinex
	go build gitlab.cs.fau.de/luksen/obinex/obinex-hwmock
	go test -c -cover gitlab.cs.fau.de/luksen/obinex/obinex-server -tags integration -o obinex-server
	go test -c -cover gitlab.cs.fau.de/luksen/obinex/obinex-watcher -tags integration -o obinex-watcher
}


function global_teardown() {
	rm obinex-hwmock obinex-server obinex-watcher obinex
	# generate coverage html
	gocovmerge server_test_*.cov > server.cov
	gocovmerge watcher_test_*.cov > watcher.cov
	gocovmerge server.cov watcher.cov > system.cov
	go tool cover -html system.cov -o coverage_system.html
	rm *_test_*.cov server.cov watcher.cov
}


success=0
global_setup
if [ -z "$@" ]
then
	bats .
	success=$?
else
	bats "$@"
	success=$?
fi
global_teardown
exit $success
