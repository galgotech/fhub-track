#!/bin/bash

BIN="go run ./cmd/fhub-track/main.go"
#BIN="./bin/fhub-track"

TEST_CLI="${BIN} -repository https://github.com/grafana/grafana.git -work-tree tmp/project"

# -init
# -track public/app/core/components/NavBar
# -status

$TEST_CLI $@

