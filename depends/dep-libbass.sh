#!/bin/bash
set -o errexit
set -o pipefail
set -x
unzip $(dirname $(readlink -f $0))/bass24-linux.zip -d  $(dirname $(readlink -f $0))/bass
unzip $(dirname $(readlink -f $0))/bass_fx24-linux.zip -d $(dirname $(readlink -f $0))/bass_fx
unzip $(dirname $(readlink -f $0))/bassmix24-linux.zip -d $(dirname $(readlink -f $0))/bassmix
cp  $(dirname $(readlink -f $0))/bass/libs/$(uname -m)/*  .
cp  $(dirname $(readlink -f $0))/bass_fx/libs/$(uname -m)/*  .
cp  $(dirname $(readlink -f $0))/bassmix/libs/$(uname -m)/*  .
rm -r $(dirname $(readlink -f $0))/bass 
rm -r $(dirname $(readlink -f $0))/bass_fx  
rm -r $(dirname $(readlink -f $0))/bassmix
