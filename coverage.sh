#!/bin/bash

mode=set
packages=(obinex-server obinex-watcher)

for dir in ${packages[*]}
do
	echo -------------------------
	echo " $dir"
	echo -------------------------
	cd $dir
	go test -covermode=$mode -coverprofile=coverage 2>/dev/null && go tool cover -html=coverage -o=../coverage_$dir.html
	rm coverage
	cd ..
done
