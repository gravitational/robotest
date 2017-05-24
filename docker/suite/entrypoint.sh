#!/bin/sh
set -e

exec robotest-suite -test.timeout 180m -test.v "$@"
