# Release Documentation

This document contains Robotest's API stability & versioning guarantees
as well as instructions to help maintainers meet these guarantees and
ensure consistent releases.

# API Stability
Robotest follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
(semver) for all image tags with a version specifier (for example `2.0.0`).
Named tags (for example `gce-stable`) offer no stability promises and may
change at any point.

Robotest's command line flags and configuration options logs are
its [semver public API](https://semver.org/spec/v2.0.0.html#spec-item-1).
This includes configuration via environment variables in the
[run_suite.sh](./docker/suite/run_suite.sh) helper script.

Notably: console output, cloud logs, and error messages are not part of
Robotest's public API and may change in minor releases.

All Go APIs (exported or not) are not part of Robotest's public API and may
change at any time.

These stability guarantees specifically target Gravity's CI use of Robotest.

Prior to 2.0.0, Robotest broke semver rules, with both new features and
backwards incompatible changes added as patch releases.  Here be dragons. :dragon:

# Release Instructions

## Prepare

### Select the commit that will become the release.

Checkout the branch/commit that will become the release.

```
# git checkout master
```

### Check that changes are within the semver risk profile.

Use `git log` to check that all changes have <= the appropriate risk for the
planned release bump (major, minor, patch). Each commit in the release should
be traceable to a PR with the "Risk Profile" filled out.

If this will be the first official release after a one or more
[pre-release](https://semver.org/spec/v2.0.0.html#spec-item-9) versions (alpha,
beta, rc, dev), make sure to audit commits back to the prior official
release.

### Create a new git tag.

Use a `v` prefix for the git tag, to play nicely with
[golang issue #32945](https://github.com/golang/go/issues/32945). Don't `v`
prefix the version elsewhere (e.g. tag annotation or release).

Signing is important to make sure the tag (and thus
release) comes from a known Gravitational maintainer.

For the rest of these instructions `2.0.0` is an example placeholder for
the version.  Replace 2.0.0 with the actual version you want to release.

```
$ git tag --sign --message "Robotest 2.0.0" v2.0.0
```

Run `make version` to double check that the build system correctly picks up
the tag.

```
$ make version
version metadata saved to version.go
Robotest Version: 2.0.0
```

### Push the tag to GitHub.

```
$ git push origin v2.0.0
Enumerating objects: 1, done.
Counting objects: 100% (1/1), done.
Writing objects: 100% (1/1), 807 bytes | 807.00 KiB/s, done.
Total 1 (delta 0), reused 0 (delta 0)
To github.com:gravitational/robotest.git
 * [new tag]         v2.0.0 -> v2.0.0
```

### Create a draft release in GitHub.

Navigate to https://github.com/gravitational/robotest/releases/tag/v2.0.0

Click Edit Tag. Enter "Robotest 2.0.0" in the "Release Title" field.

Add a concise note about what the release contains in the "Describe this
release" field.

Click "Save Draft".

## Release
With all the preperation taken care of, publishing the release will take only
a minute or two, making the artifacts and release history publicly available.

### Run the "Robotest-publish" Jenkins job.

Navigate to: https://jenkins.gravitational.io/view/Robotest/job/Robotest-publish/

Click "Build with Parameters".

Enter your tag into the "GIT_REF" field, prefixed with "tags/" e.g. `tags/v2.0.0`

Ignore Robotest-publish's TAG parameter. This is unneeded when an annotated
git tag is used.

Build!

After the build & publish completes successfully verify the new image tag
is present in https://quay.io/repository/gravitational/robotest-suite.

### Publish the draft GitHub release.
After the new quay.io artifact is available, publish the draft release
created earlier in the Prepare step.

Congratulations! Robotest is released!
