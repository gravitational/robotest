NAME := robotest-standalone
TARBALL_NAME := $(NAME).tar
VER ?= 0.0.1
IMAGE := quay.io/gravitational/$(NAME):$(VERSION)

# Amazon S3
ifeq (, $(shell which aws))
	$(error "No aws cli in $(PATH)")
endif
BUILD_BUCKET_URL := s3://clientbuilds.gravitational.io/gravity/latest
S3_OPTS := --region us-east-1

#
# build an image with standalone headless chromedriver server
#
build:
	docker build -t $(IMAGE) .

#
# run chromedriver service locally for development/testing
#
run: build
	-docker rm -f chromedriver
	docker run -u 995 -d -p 9515:9515 --name=$(NAME) $(IMAGE)

#
# save builded image into tarball
#
save:
	docker save --output $(TARBALL_NAME) $(IMAGE)

#
# publish tarball with standalone robotest
#
publish:
	docker push $(IMAGE)
	aws s3 cp $(NAME).tar $(BUILD_BUCKET_URL)/$(TARBALL_NAME)
