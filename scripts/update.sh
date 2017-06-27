#!/bin/bash

set -e                                                                          
                                                                                
shell="sudo -E -u i4obinex bash -c"                                             
                                                                                
cd src/gitlab.cs.fau.de/luksen/obinex/                                          
$shell "git pull"                                                               
$shell "git submodule init"                                                     
$shell "git submodule update"                                                   
export GOPATH=/proj/i4obinex/system                                             
$shell "go install gitlab.cs.fau.de/luksen/obinex/..."            
