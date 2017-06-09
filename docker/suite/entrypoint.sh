#!/bin/sh

robotest-suite -test.timeout 180m -test.v "$@" 2>&1 | tee ${JUNIT_REPORT}.txt || ROBOTEST_RESULT=$?
go-junit-report ${JUNIT_REPORT}.txt > ${JUNIT_REPORT}
exit ${ROBOTEST_RESULT}