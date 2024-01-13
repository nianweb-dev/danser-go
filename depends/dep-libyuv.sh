#!/bin/bash
if [ -e /usr/lib/$(uname -m)-linux-gnu/libyuv.so.0 ]  
    then 
        echo found libyuv0 at $(readlink -f /usr/lib/$(uname -m)-linux-gnu/libyuv.so.0 )
        cp $(readlink -f /usr/lib/$(uname -m)-linux-gnu/libyuv.so.0 ) ./libyuv.so
    else 
        echo libyuv0 not found. 
        exit 1
    fi