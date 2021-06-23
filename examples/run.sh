#!/bin/bash

go run receiver/main.go --once --cpuprofile receiver.prof &

sleep .5

go run sender/main.go

wait