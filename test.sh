#/bin/sh

set -e
coverage_script="$1"

-rm -f *.txt

# ./... ignores vendor: https://golang.org/doc/go1.9#vendor-dotdotdot
go test -race -coverprofile="coverage.txt" ./...

if [ -n "$coverage_script" ]; then
	echo "Uploading test coverage results using script: $coverage_script"
	# this expects the .travis.yml to setup https://codecov.io/bash locally
	(eval "$coverage_script")
fi

