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
it's [semver public API](https://semver.org/spec/v2.0.0.html#spec-item-1).
This includes configuration via environment variables in the
[run_suite.sh](./docker/suite/run_suite.sh) helper script.

Notably: console output, cloud logs, and error messages are not part of
Robotest's public API and may change in minor releases.

All go APIs (exported or not) are not part of Robotest's public API and may
change at any time.

These stability guarantees specifically target's Gravity's CI use of Robotest.

Prior to 2.0.0, Robotest broke semver rules, with both new features and
backwards incompatible changes added as patch releases.  Here be dragons. :dragon:
