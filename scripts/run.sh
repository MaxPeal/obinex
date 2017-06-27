#!/bin/bash

shell="sudo -u i4obinex bash -c"                                                
                                                                                
# run once for credentials                                                      
$shell ""                                                                       
                                                                                
case $HOSTNAME in                                                               
"i4jenkins")                                                                    
›       $shell "bin/obinex-watcher -servers faui49jenkins15 2> watcher.log" &   
;;                                                                              
"faui49jenkins15")                                                              
›       $shell "bin/obinex-server 2> fastbox.log" &                             
;;                                                                              
*)                                                                              
exit                                                                            
;;                                                                              
esac                                                                            
