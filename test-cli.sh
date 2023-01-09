#!/bin/bash

BIN="go run ./cmd/fhub-track/main.go"

TEST_CLI="${BIN} --src tmp/grafana --dst tmp/project"

$TEST_CLI $@
