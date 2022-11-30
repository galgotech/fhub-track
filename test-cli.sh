#!/bin/bash

BIN="go run ./cmd/fhub-track/main.go"
#BIN="./bin/fhub-track"

TEST_CLI="${BIN} -work-tree-src tmp/grafana -work-tree-dst tmp/project"

# -init
# -track public/app/core/components/NavBar
# -status

$TEST_CLI $@
