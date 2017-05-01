#!/bin/sh
export DISPLAY=:10
Xvfb ${DISPLAY} -screen 0 1366x768x24 +extension RANDR -ac &
service dbus restart
chromedriver  --whitelisted-ips="" &
robotest 2>&1 $@
