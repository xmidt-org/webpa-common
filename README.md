# webpa-common

[![Build Status](https://travis-ci.org/Comcast/webpa-common.svg?branch=master)](https://travis-ci.org/Comcast/webpa-common) 
[![codecov.io](http://codecov.io/github/Comcast/webpa-common/coverage.svg?branch=master)](http://codecov.io/github/Comcast/webpa-common?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/Comcast/webpa-common)](https://goreportcard.com/report/github.com/Comcast/webpa-common)

## Environment Setup Instructions

**Assumptions:**
  - Go with version >= 1.10 https://golang.org/dl/
  - Latest version of Glide https://github.com/Masterminds/glide


**1)** Set up a new workspace (Optional, skip to step 2 if you want to edit webpa-common in your existing workspace)
```
newWorkSpace=~/xmidt   #this can be any path you want
export GOPATH=$newWorkSpace
```
**2)** Create necessary path
```
mkdir -p $GOPATH/github.com/Comcast
```
**3)** Clone repo
 ```
 cd $GOPATH/github.com/Comcast
 git clone git@github.com:Comcast/webpa-common.git
 ```
**4)** Get Dependencies
 ```
 cd webpa-common
 glide install --strip-vendor
 ```
 
**5)** Try running the tests!
  ```
  ./test.sh
  ```
  
