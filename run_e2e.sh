export P='/Users/denismishin/src/gravi/robotest' 
docker run \
	-v ${P}/wd/gravity:/usr/bin/gravity \
	-v ${P}/wd:/robotest \
	-v ${P}/build/robotest-e2e:/usr/bin/robotest-e2e \
	-v ${P}/assets/terraform:/robotest/terraform \
	quay.io/gravitational/robotest-e2e:1.0.35 \
	-config /robotest/config/config.azure.yaml \
	-ginkgo.focus="Onprem Installation" -debug -mode=wizard