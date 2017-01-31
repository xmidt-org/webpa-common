#/bin/sh

set -e

for package in $(go list ./... | grep -v vendor); do
	go test -race -coverprofile=coverage.txt $package

	# this expects the .travis.yml to setup https://codecov.io/bash locally
	./codecov.sh
done
