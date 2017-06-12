#!/bin/sh

robotest-suite -test.timeout 48h -test.v "$@" 2>&1 | tee ${JUNIT_REPORT}.txt || ROBOTEST_RESULT=$?
cat ${JUNIT_REPORT}.txt | go-junit-report  > ${JUNIT_REPORT}
exit ${ROBOTEST_RESULT}
