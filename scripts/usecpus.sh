#!/bin/sh

CPU_CORES=8 # MY personal number of CPU cores

if [ -z $1 ]; then
    echo "Usage: usecpus.sh <number of cpus>"
    exit 1
fi

#if ! [ -f /.dockerenv ]; then
#    echo "Not inside a container!";
#   exit 1
#fi

cpus=$1

if [ $cpus -lt 1 ]; then
    echo "Number of cores must be >=1"
    exit 1
fi

for i in $(seq 0 $(expr $cpus - 1) ); do
    echo enable $i;
    echo 1 > /sys/devices/system/cpu/cpu${i}/online
done

for i in $(seq $cpus $(expr $CPU_CORES - 1) ); do
    echo disable $i;
    echo 0 > /sys/devices/system/cpu/cpu${i}/online
done
