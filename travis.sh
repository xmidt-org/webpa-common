export DIRLIST=`find . -type f -name '*.go' -exec dirname {} \; | sort -u`

for DIR in $DIRLIST; do
	pushd $DIR
	go test -coverprofile=coverage.txt
	popd
done