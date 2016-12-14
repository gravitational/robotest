NAME := selenium
VER ?= 0.0.1

#
# build an image with standalone headless selenium server
#
build:
	docker build -t $(NAME):$(VER) .

#
# run selenium service locally for development/testing
#
run: build
	-docker rm -f selenium
	docker run -u 995 -d -p 9515:9515 --name=$(NAME) $(NAME):$(VER)

#
# deploy to build server (the docker image is built on build server too)
#
deploy:
	ansible-playbook -i hosts deploy.yaml -v
