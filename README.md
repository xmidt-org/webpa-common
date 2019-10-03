# webpa-common

[![Build Status](https://travis-ci.org/xmidt-org/webpa-common.svg?branch=master)](https://travis-ci.org/xmidt-org/webpa-common) 
[![codecov.io](http://codecov.io/github/xmidt-org/webpa-common/coverage.svg?branch=master)](http://codecov.io/github/xmidt-org/webpa-common?branch=master)
[![Code Climate](https://codeclimate.com/github/xmidt-org/webpa-common/badges/gpa.svg)](https://codeclimate.com/github/xmidt-org/webpa-common)
[![Issue Count](https://codeclimate.com/github/xmidt-org/webpa-common/badges/issue_count.svg)](https://codeclimate.com/github/xmidt-org/webpa-common)
[![Go Report Card](https://goreportcard.com/badge/github.com/xmidt-org/webpa-common)](https://goreportcard.com/report/github.com/xmidt-org/webpa-common)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](https://github.com/xmidt-org/webpa-common/blob/master/LICENSE)
[![GitHub release](https://img.shields.io/github/release/xmidt-org/webpa-common.svg)](CHANGELOG.md)

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
mkdir -p $GOPATH/github.com/xmidt-org
```
**3)** Clone repo
 ```
 cd $GOPATH/github.com/xmidt-org
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
  
