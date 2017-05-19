export P='/Users/denismishin/src/gravi/robotest' 
docker run \
	-v ${P}/wd_suite:/robotest \
	-v ${P}/build/robotest-suite:/usr/bin/robotest-suite \
	-v ${P}/assets/terraform:/robotest/terraform \
	quay.io/gravitational/robotest-suite:1.0.35 \
	-config /robotest/config/config.yaml \
	-dir=/robotest/state -tag=robotest-west-1