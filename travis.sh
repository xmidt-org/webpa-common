#/bin/sh

set -e

for package in $(go list ./... | grep -v vendor); do
	echo "*********************************************************"
	echo "** $package"
	echo "*********************************************************"
	go test -race -coverprofile=coverage.txt $package

	# this expects the .travis.yml to setup https://codecov.io/bash locally
	./codecov.sh

	# remove coverage.txt after each run, to avoid uploading coverage for packages without tests
	rm coverage.txt
done
