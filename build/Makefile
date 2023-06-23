IMAGE_NAME = jerriedr-builder

.PHONY: build
build:
	docker build --platform=linux/amd64 -t $(IMAGE_NAME) .

.PHONY: test
test:
	docker build --platform=linux/amd64 -t $(IMAGE_NAME)-candidate .
	IMAGE_NAME=$(IMAGE_NAME)-candidate test/run
