# webpa-common

[![Build Status](https://github.com/xmidt-org/webpa-common/workflows/CI/badge.svg)](https://github.com/xmidt-org/webpa-common/actions)
[![codecov.io](http://codecov.io/github/xmidt-org/webpa-common/coverage.svg?branch=main)](http://codecov.io/github/xmidt-org/webpa-common?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/xmidt-org/webpa-common)](https://goreportcard.com/report/github.com/xmidt-org/webpa-common)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](https://github.com/xmidt-org/webpa-common/blob/main/LICENSE)
[![GitHub release](https://img.shields.io/github/release/xmidt-org/webpa-common.svg)](CHANGELOG.md)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=xmidt-org_webpa-common&metric=alert_status)](https://sonarcloud.io/dashboard?id=xmidt-org_webpa-common)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/xmidt-org/webpa-common)](https://pkg.go.dev/github.com/xmidt-org/webpa-common)

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Environment Requirements](#environment-requirements)
- [Testing The Library](#testing-the-library)
- [Contributing](#contributing)

## Code of Conduct

This project and everyone participating in it are governed by the [XMiDT Code Of Conduct](https://xmidt.io/code_of_conduct/). 
By participating, you agree to this Code.

## Environment Requirements

  - Go with version >= 1.12 https://golang.org/dl/

## Testing the Library

To run the tests, `git clone` the repository, then from within the repo directory run:
  ```
  go test ./... -race -coverprofile=coverage.txt
  ```

## Contributing

Refer to [CONTRIBUTING.md](CONTRIBUTING.md).  
