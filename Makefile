GCE_APP=cable
ifeq ($(GCE_PROJECT_ID),)
GCE_PROJECT_ID := $(GC_APP)
endif
IMAGE_NAME=gcr.io/$(GCE_PROJECT_ID)/$(GCE_APP):latest
SERVER_HOST_PORT=8080
SERVER_CONTAINER_PORT=8080
SERVER_PORT_MAPPING=$(SERVER_HOST_PORT):$(SERVER_CONTAINER_PORT)

.PHONY: build
build:
	docker build -t $(IMAGE_NAME) .

.PHONY: check
check:
	curl --write-out %{http_code} --silent --output /dev/null localhost:$(SERVER_HOST_PORT)/_health	

.PHONY: deploy
deploy: build
	gcloud docker -- push $(IMAGE_NAME)
	gcloud beta run deploy $(GCE_APP) --image $(IMAGE_NAME) --platform managed --region us-central1

.PHONY: logs
logs:
	docker logs -f $$(docker ps -a -q --filter ancestor="$(IMAGE_NAME)" --format="{{.ID}}")

.PHONY: start
start: build
	docker run --rm -d -p $(SERVER_PORT_MAPPING) $(IMAGE_NAME)

.PHONY: stop
stop:
	docker stop $$(docker ps -a -q --filter ancestor="$(IMAGE_NAME)" --format="{{.ID}}")

.PHONY: test
test:
	go test ./...

.PHONY: test
coverage:
	go test ./... -coverprofile coverage.out
	go tool cover -html coverage.out