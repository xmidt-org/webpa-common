Contribution Guidelines
=======================

We love to see contributions to the project and have tried to make it easy to 
do so. If you would like to contribute code to this project you can do so 
through GitHub by [forking the repository and sending a pull request](http://gun.io/blog/how-to-github-fork-branch-and-pull-request/).

Before Comcast merges your code into the project you must sign the 
[Comcast Contributor License Agreement (CLA)](https://gist.github.com/ComcastOSS/a7b8933dd8e368535378cda25c92d19a).

If you haven't previously signed a Comcast CLA, you'll automatically be asked 
to when you open a pull request. Alternatively, we can e-mail you a PDF that 
you can sign and scan back to us. Please send us an e-mail or create a new 
GitHub issue to request a PDF version of the CLA.

If you have a trivial fix or improvement, please create a pull request and 
request a review from a [maintainer](MAINTAINERS.md) of this repository.

If you plan to do something more involved, that involves a new feature or 
changing functionality, please first create an [issue](#issues) so a discussion of 
your idea can happen, avoiding unnecessary work and clarifying implementation.

A relevant coding style guideline is the [Go Code Review Comments](https://code.google.com/p/go-wiki/wiki/CodeReviewComments).

Documentation
-------------

If you contribute anything that changes the behavior of the application, 
document it in the follow places as applicable:
* the code itself, through clear comments and unit tests
* [README](README.md)

This includes new features, additional variants of behavior, and breaking 
changes.

Testing
-------

Tests are written using golang's standard testing tools, and are run prior to 
the PR being accepted.

Issues
------

For creating an issue:
* **Bugs:** please be as thorough as possible, with steps to recreate the issue 
  and any other relevant information.
* **Feature Requests:** please include functionality and use cases.  If this is 
  an extension of a current feature, please include whether or not this would 
  be a breaking change or how to extend the feature with backwards 
  compatibility.
* **Security Vulnerability:** please report it at 
  https://my.xfinity.com/vulnerabilityreport and contact the [maintainers](MAINTAINERS.md).

If you wish to work on an issue, please assign it to yourself.  If you have any
questions regarding implementation, feel free to ask clarifying questions on 
the issue itself.

Pull Requests
-------------

* should be narrowly focused with no more than 3 or 4 logical commits
* when possible, address no more than one issue
* should be reviewable in the GitHub code review tool
* should be linked to any issues it relates to (i.e. issue number after (#) in commit messages or pull request message)
* should conform to idiomatic golang code formatting
