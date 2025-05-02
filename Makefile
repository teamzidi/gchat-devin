API := gchat-devin
PORT := 8080

PROJECT := $(shell gcloud config get-value project)
GROUP := gchat
SERVICE := $(GROUP)-devin
REGION := asia-northeast1
SERVICE_ACCOUNT := $(SERVICE)@$(PROJECT).iam.gserviceaccount.com

VERSION := $(shell git describe --tags || echo dev)
SUFFIX = $(subst .,-,$(VERSION))
IMAGE := $(REGION)-docker.pkg.dev/$(PROJECT)/$(GROUP)/devin:$(VERSION)

DEVIN_API_KEY_SECRET := devin-api-key:latest
GOOGLE_CHAT_API_SECRET := google-chat-api-key:latest

export PORT
export PROJECT

.PHONY: usage run dev deploy build publish check_version test lint

usage:
	@echo "See Makefile"

run:
	go run .

dev:
	go run -tags=develop .

lint:
	golangci-lint run --config .golangci.yml
	- grep --exclude=Makefile --color -R -E '(FIXME|TODO|NOTE)' *

test:
	go test -race -run=$(TEST) ./...

deploy: build publish
	gcloud --project=$(PROJECT) run deploy $(SERVICE) \
		--revision-suffix=$(SUFFIX) \
		--region=$(REGION) \
		--cpu=1 \
		--memory=256Mi \
		--max-instances=10 \
		--concurrency=80 \
		--network=default \
		--subnet=default \
		--allow-unauthenticated \
		--set-env-vars=PROJECT="$(PROJECT)" \
		--set-secrets=DEVIN_API_KEY=$(DEVIN_API_KEY_SECRET) \
		--set-secrets=GOOGLE_CHAT_API_KEY=$(GOOGLE_CHAT_API_SECRET) \
		--service-account=$(SERVICE_ACCOUNT) \
		--image=$(IMAGE)

build: check_version
	docker buildx build --platform=linux/amd64 -t $(IMAGE) .

publish: check_version
	docker push $(IMAGE)

check_version:
	@if [[ "$(VERSION)" =~ (dev|^[0-9]+\.[0-9]+\.[0-9]+(-beta.[0-9]+)?)$$ ]]; then \
		echo "VERSION:" $(VERSION); \
	else \
		echo "$(VERSION) is invalid version format."; \
		exit 1; \
	fi
