####################################################################################################
# builder
####################################################################################################
FROM golang:1.12.5-stretch as builder

RUN apt-get update -q && apt-get install -yq --no-install-recommends \
    git \
    make \
    wget \
    gcc \
    zip \
    bzip2 \
    lsb-release \
    software-properties-common \
    apt-transport-https \
    ca-certificates \
    vim \
    && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

WORKDIR /src

# Install golangci-lint
ENV GOLANGCI_LINT_VERSION=1.16.0
RUN curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/v$GOLANGCI_LINT_VERSION/install.sh| sh -s -- -b $(go env GOPATH)/bin v$GOLANGCI_LINT_VERSION

# Install github-release
ENV GITHUB_RELEASE_VERSION=0.7.2
RUN curl -sLo- https://github.com/aktau/github-release/releases/download/v${GITHUB_RELEASE_VERSION}/linux-amd64-github-release.tar.bz2 | \
    tar -xjC "$GOPATH/bin" --strip-components 3 -f-

COPY go.mod .
COPY go.sum .
RUN go mod download

# Perform the build
COPY . .
RUN make build

####################################################################################################
# runtime
####################################################################################################
FROM ubuntu:16.04 as runtime

RUN apt-get update -q && apt-get install -yq --no-install-recommends ca-certificates && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

COPY --from=builder /src/dist/* /usr/local/bin/

####################################################################################################
# agent-node-attestor
####################################################################################################
FROM runtime as node-attestor
ENTRYPOINT ["/usr/local/bin/oidc_node_attestor"]

####################################################################################################
# workload-attestor
####################################################################################################
FROM runtime as workload-attestor
ENTRYPOINT ["/usr/local/bin/oidc_workload_attestor"]

