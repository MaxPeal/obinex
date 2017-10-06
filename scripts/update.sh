#!/usr/bin/env bash

set -e

shell="sudo -E -u i4obinex bash -c"

cd src/gitlab.cs.fau.de/i4/obinex/
$shell "git pull"
$shell "git submodule update --init --recursive"
export GOPATH=/proj/i4obinex/system
$shell "go install gitlab.cs.fau.de/i4/obinex/..."
