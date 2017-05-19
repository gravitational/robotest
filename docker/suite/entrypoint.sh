#!/bin/sh
set -e

exec robotest-suite -test.timeout 60m -test.v "$@"
