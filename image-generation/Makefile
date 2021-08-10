BUILDER_IMG ?= ""

docker: docker-build docker-push

docker-build:
	docker build -t ${BUILDER_IMG} .

docker-push:
	docker push ${BUILDER_IMG}