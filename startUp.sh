#!/bin/bash

bg go run ./Broker/broker.go
bg go run ./worker/worker.go
bg go run ./worker/worker.go
bg go run ./worker/worker.go

go run .

