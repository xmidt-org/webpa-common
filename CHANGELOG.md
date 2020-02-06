# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [v1.6.3]
- added session-id to request response message [#454](https://github.com/xmidt-org/webpa-common/pull/454)

## [v1.6.2]
- Updated capabilityChecker to include metrics and configure whether or not to return errors [#449](https://github.com/xmidt-org/webpa-common/pull/449)

## [v1.6.1]
- Fixed panic from assignment to entry in nil map [#453](https://github.com/xmidt-org/webpa-common/pull/453)

## [v1.6.0]
- Added session-id to device information [#451](https://github.com/xmidt-org/webpa-common/pull/451)
- Added device metadata to outbound events [#451](https://github.com/xmidt-org/webpa-common/pull/451)

## [v1.5.1]
- Added automated releases using travis [#444](https://github.com/xmidt-org/webpa-common/pull/444)
- Bumped bascule version to v0.7.0 and updated basculechecks to match [#444](https://github.com/xmidt-org/webpa-common/pull/444)

## [v1.5.0]
- reduced logging for xhttp retry #441
- modified capabilities check to be more restrictive #440

## [v1.4.0]
- Moved from glide to go modules
- Updated bascule version to v0.5.0
- Updated wrp-go to v1.3.3
- Updated README to match go modules
- No longer accept retries in webhook.W

## [v1.3.2]
- Bump Bascule to v0.2.7

## [v1.3.1]
- Downgraded bascule version

## [v1.3.0]
- removed `wrp` package

## [v1.2.0]
- Added minVersion and maxVersion to `server` package.
- Added cpuprofile and memprofile flags.
- Updated import paths.


## [v1.1.0]
- Added ability to check status code for retrying http request
- Added ability to update http.Request for `xhttp` retry
- Added MaxRetry and AlternativeURLS for update webhook config

## [v1.0.1]
- Fix for https://github.com/Comcast/webpa-common/issues/364
- Removed unused dep files.
- Added capability checks to be used when consuming `bascule` package.
- Fix for responseRequest test that was intermittently failing.

## [v1.0.0]
 - The first official release. We will be better about documenting changes 
   moving forward.

[Unreleased]: https://github.com/xmidt-org/webpa-common/compare/v1.6.3...HEAD
[v1.6.3]: https://github.com/xmidt-org/webpa-common/compare/v1.6.2...v1.6.3
[v1.6.2]: https://github.com/xmidt-org/webpa-common/compare/v1.6.1...v1.6.2
[v1.6.1]: https://github.com/xmidt-org/webpa-common/compare/v1.6.0...v1.6.1
[v1.6.0]: https://github.com/xmidt-org/webpa-common/compare/v1.5.1...v1.6.0
[v1.5.1]: https://github.com/xmidt-org/webpa-common/compare/v1.5.0...v1.5.1
[v1.5.0]: https://github.com/xmidt-org/webpa-common/compare/v1.4.0...v1.5.0
[v1.4.0]: https://github.com/xmidt-org/webpa-common/compare/v1.3.2...v1.4.0
[v1.3.2]: https://github.com/xmidt-org/webpa-common/compare/v1.3.1...v1.3.2
[v1.3.1]: https://github.com/xmidt-org/webpa-common/compare/v1.3.0...v1.3.1
[v1.3.0]: https://github.com/xmidt-org/webpa-common/compare/v1.2.0...v1.3.0
[v1.2.0]: https://github.com/xmidt-org/webpa-common/compare/v1.1.0...v1.2.0
[v1.1.0]: https://github.com/xmidt-org/webpa-common/compare/v1.0.1...v1.1.0
[v1.0.1]: https://github.com/xmidt-org/webpa-common/compare/v1.0.0...v1.0.1
[v1.0.0]: https://github.com/xmidt-org/webpa-common/compare/v0.9.0-alpha...v1.0.0

