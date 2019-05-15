# Enable go modules
export GO111MODULE=on

# Make variables
NAME        := spire-oidc-attestor-plugin
PROJECTROOT := $(shell pwd)
VERSION     ?= $(shell cat ${PROJECTROOT}/VERSION)-dev
REVISION    := $(shell git rev-parse --short HEAD)
IMAGE_PREFIX ?= everpeace/
IMAGE_TAG   ?= $(VERSION)
OUTDIR      ?= $(PROJECTROOT)/dist
RELEASE_TAG ?=
GITHUB_USER := everpeace
GITHUB_REPO := spire-oidc-attestor-plugin
GITHUB_REPO_URL := git@github.com:$(GITHUB_USER)/$(GITHUB_REPO).git
GITHUB_TOKEN ?=
binary_dirs := $(shell cd cmd && find */* -maxdepth 0 -type d)
LDFLAGS := -ldflags="-s -w -X \"github.com/everpeace/spire-oidc-attestor-plugin/pkg.Version=$(VERSION)\" -X \"github.com/everpeace/spire-oidc-attestor-plugin/pkg.Revision=$(REVISION)\" -extldflags \"-static\""


guard-%:
	@ if [ "${${*}}" = "" ]; then \
    echo "Environment variable $* is not set"; \
		exit 1; \
	fi

.PHONY: build
build: $(binary_dirs)

$(binary_dirs):
	cd cmd/$@ && go build -tags netgo -installsuffix netgo $(LDFLAGS) -o $(OUTDIR)/$@

.PHONY: build-linux-amd64
build-linux-amd64:
	make build \
		GOOS=linux \
		GOARCH=amd64 \
		NAME=spire-oidc-attestor-plugin-linux-amd64

.PHONY: build-linux
build-linux: build-linux-amd64

.PHONY: build-darwin
build-darwin:
	make build \
		GOOS=darwin \
		NAME=spire-oidc-attestor-plugin-darwin-amd64

.PHONY: build-windows
build-windows:
	make build \
		GOARCH=amd64 \
		GOOS=windows \
		NAME=spire-oidc-attestor-plugin-windows-amd64.exe

.PHONY: build-all
build-all: build-linux build-darwin build-windows

.PHONY: build-node-attestor-image
build-node-attestor-image:
	docker build -t $(IMAGE_PREFIX)spire-oidc-node-attestor-plugin:$(IMAGE_TAG) --target node-attestor $(PROJECTROOT)

.PHONY: build-workload-attestor-image
build-workload-attestor-image:
	docker build -t $(IMAGE_PREFIX)spire-oidc-workload-attestor-plugin:$(IMAGE_TAG) --target workload-attestor $(PROJECTROOT)

.PHONY: release
release: release-code release-assets release-image

.PHONY: release-code
release-code: guard-RELEASE_TAG guard-RELEASE_COMMIT guard-GITHUB_USER guard-GITHUB_REPO guard-GITHUB_REPO_URL guard-GITHUB_TOKEN
	@GITHUB_TOKEN=$(GITHUB_TOKEN)
	git tag $(RELEASE_TAG) $(RELEASE_COMMIT)
	git push $(GITHUB_REPO_URL) $(RELEASE_TAG)
	github-release release \
	  --user $(GITHUB_USER) \
		--repo $(GITHUB_REPO) \
		--tag $(RELEASE_TAG)

.PHONY: release-assets
release-assets: guard-RELEASE_TAG guard-RELEASE_COMMIT guard-GITHUB_USER guard-GITHUB_REPO guard-GITHUB_REPO_URL guard-GITHUB_TOKEN
	@GITHUB_TOKEN=$(GITHUB_TOKEN)
	git diff --quiet HEAD || (echo "your current branch is dirty" && exit 1)
	git checkout $(RELEASE_COMMIT)
	make clean build-all VERSION=$(shell cat ${PROJECTROOT}/VERSION)
	for target in linux-amd64 darwin-amd64 windows-amd64.exe; do \
		github-release upload \
		  --user $(GITHUB_USER) \
			--repo $(GITHUB_REPO) \
			--tag $(RELEASE_TAG) \
			--name spire-oidc-attestor-plugin-$$target \
			--file $(OUTDIR)/spire-oidc-attestor-plugin-$$target; \
	done
	git checkout -

.PHONY: release-image
release-image: IMAGE_TAG=$(RELEASE_TAG)
release-image: guard-RELEASE_TAG
	git diff --quiet HEAD || (echo "your current branch is dirty" && exit 1)
	git checkout $(RELEASE_COMMIT)
	make build-image
	docker push $(IMAGE_PREFIX)spire-oidc-attestor-plugin-cli:$(RELEASE_TAG)
	git checkout -

.PHONY: lint
lint:
	golangci-lint run --config golangci.yml

test:
	@go test -v -race -short -tags no_e2e ./cmd/... ./pkg/...

.PHONY: e2e
e2e:
	@go test -v $(PROJECTROOT)/test/e2e/e2e_test.go

.PHONY: coverage
coverage:
	@go test -tags no_e2e -covermode=count -coverprofile=profile.cov -coverpkg ./pkg/...,./cmd/... $(shell go list ./... | grep -v /vendor/)
	@go tool cover -func=profile.cov

.PHONY: clean
clean:
	rm -rf $(OUTDIR)/*