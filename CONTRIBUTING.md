# Contributing to Robotest

Robotest is an open source project.

Robotest is the work of [many contributors](https://github.com/gravitational/robotest/graphs/contributors).
We appreciate your help! :tada:

Robotest contributors follow a [code of conduct](./CODE_OF_CONDUCT.md).

Robotest tends to have mostly Gravitational staff as contributors, so
we're excited to get any outside contributions, including issues.

## Filing an Issue

Security issues should be reported directly to security@gravitational.com.

If you are unsure if you've found a bug, please search Robotest's
[current issues](https://github.com/gravitational/robotest/issues).

## Contributing A Patch

If you're working on an existing issue, respond to the issue and express
interest in working on it. This helps other contributors know that the issue is
active, and hopefully prevents duplicated efforts.

If you want to work on a new idea:

1. Submit an issue describing the proposed change and the implementation.
2. Robotest maintainers will triage & respond to your issue.
3. Write your code, test your changes. Run `make lint`, `make test` and `make build`.
Most importantly,  _communicate_ with the maintainers as you develop.
4. Submit a pull request from your fork.

# Coding Guidelines

## Adding dependencies

If you wish to add new dependencies (anything that touches `Dockerfile`s or `go.mod`),
the new dependencies must be:

- licensed via Apache2 license
- approved by Gravitational staff

## Compatibility & API Stability

Robotest follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
(semver) as described in the [release documentation](./RELEASE.md)

Due to semver's strict requirements, patches that introduce new options
or change behavior of the public API may need to wait for the next major or minor
release for merge. Major & minor releases are issued on an as needed basis.

## Style

Robotest follows [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments),
in alignment with all of Gravitational's go code.  Some (but not all) of the
guidelines are automated by Robotest's `make lint` target.  Please run
`make lint` before submitting code to save reviewers the hassle of stylistic
nits.
