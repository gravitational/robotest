# Robotest


Robotest is a tool for running automated integration tests against [Gravity](https://gravitational.com/gravity).

# Running
Robotest is distributed as a [Open Container Initiative](https://opencontainers.org/)
image at https://quay.io/repository/gravitational/robotest-suite.

To learn how to invoke `robotest-suite`, see the [suite README](./suite/README.md).

# Upgrading
To change to a different version of Robotest, simply depend on a different
image tag from quay.io.

## API Stability
See Robotest's [release documentation](./RELEASE.md) for information about
API stability and backwards compatibility between Robotest versions.

# Contributing
If you would like to contribute to Robotest, check out our [contribution guidelines](CONTRIBUTING.md).

# End-to-End Testing (deprecated)
In earlier versions, Robotest offered automated Web UI based tests against the
Gravity web installer.  After [significant changes](https://github.com/gravitational/gravity/pull/424/)
to the Web UI in Gravity 6.0 these browser based tests were never updated and
have now fallen into disuse.  For more information on browser based tests see
the [e2e README](./e2e/README.md).
