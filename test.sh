#/bin/sh

set -e
test_number=1
coverage_script="$1"

rm -f *.txt

for package in $(go list ./... | grep -v vendor); do
	go test -race -coverprofile="coverage-$test_number.txt" $package &
	test_number=`expr $test_number + 1`
done

wait

if [ -n "$coverage_script" ]; then
	echo "Uploading test coverage results using script: $coverage_script"
	# this expects the .travis.yml to setup https://codecov.io/bash locally
	(eval "$coverage_script")
fi

