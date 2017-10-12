#!/bin/bash


./webinterface &

sh spider.sh -port 6882 &
sh spider.sh -port 6883	&
sh spider.sh -port 6884 &
sh spider.sh -port 6885 &

