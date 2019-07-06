#!/usr/bin/env bash

# because things are never simple.
# See https://github.com/codecov/example-go#caveat-multiple-files

set -e
echo "" > coverage.txt

# show the network configuration. This can help troubleshooting integration
# tests.
ip a

for d in $(go list ./... | grep -v vendor); do
    go test -race -coverprofile=profile.out -covermode=atomic $d
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
    # integration tests
    go test -c -cover -tags=integration -race -covermode=atomic $d
    testbin="./$(basename $d).test"
    # only run it if it was built - i.e. if there are integ tests
    test -x "${testbin}" && sudo "./${testbin}" -test.coverprofile=profile.out
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm -f profile.out
    fi
done
