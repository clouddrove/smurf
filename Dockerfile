# Stage 1: Build Go application
FROM golang:1.26-alpine AS builder

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

RUN apk add --no-cache git
WORKDIR /app
COPY . .
RUN go build -ldflags "-X 'github.com/clouddrove/smurf/cmd.version=${VERSION}' -X 'github.com/clouddrove/smurf/cmd.commit=${COMMIT}' -X 'github.com/clouddrove/smurf/cmd.date=${DATE}'" -o smurf main.go

# Stage 2: Create minimal runtime image
FROM alpine:3.18

# Install only essential CLI tools.
# python3 is listed explicitly because the Google Cloud CLI needs it: it
# arrives today only as a transitive dependency of aws-cli, which would take
# gcloud down with it if that ever changed.
RUN apk add --no-cache \
    bash \
    curl \
    unzip \
    git \
    docker-cli \
    aws-cli \
    python3

# Install Google Cloud SDK (minimal install).
# Previously this piped https://sdk.cloud.google.com straight into bash, which
# runs whatever that endpoint serves at build time, unversioned and unchecked.
# Now a pinned release tarball is fetched and its sha256 verified before
# anything from it executes. The archive unpacks to a google-cloud-sdk/ root,
# so the install dir layout and the PATH below are unchanged.
ENV CLOUDSDK_INSTALL_DIR=/usr/local/gcloud
ARG GCLOUD_VERSION=583.0.0
ARG GCLOUD_SHA256=84c5e4798836bda13aa82c3e84fa1acd0c4e4ca5318f7141052e3a5a26a7cc97
RUN curl -fsSL --proto '=https' --proto-redir '=https' --tlsv1.2 -o gcloud.tar.gz \
      "https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-${GCLOUD_VERSION}-linux-x86_64.tar.gz" && \
    echo "${GCLOUD_SHA256}  gcloud.tar.gz" | sha256sum -c - && \
    mkdir -p ${CLOUDSDK_INSTALL_DIR} && \
    tar -xzf gcloud.tar.gz -C ${CLOUDSDK_INSTALL_DIR} && \
    rm gcloud.tar.gz && \
    ${CLOUDSDK_INSTALL_DIR}/google-cloud-sdk/install.sh --quiet --usage-reporting=false --path-update=false && \
    rm -rf ${CLOUDSDK_INSTALL_DIR}/google-cloud-sdk/.install/.backup

ENV PATH $PATH:${CLOUDSDK_INSTALL_DIR}/google-cloud-sdk/bin

# Install GKE plugin
RUN gcloud components install gke-gcloud-auth-plugin --quiet

# Install Terraform. Checksum from the published terraform_1.5.7_SHA256SUMS.
ARG TERRAFORM_VERSION=1.5.7
ARG TERRAFORM_SHA256=c0ed7bc32ee52ae255af9982c8c88a7a4c610485cf1d55feeb037eab75fa082c
RUN curl -fsSL --proto '=https' --proto-redir '=https' --tlsv1.2 -o terraform.zip \
      "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip" && \
    echo "${TERRAFORM_SHA256}  terraform.zip" | sha256sum -c - && \
    unzip terraform.zip && \
    mv terraform /usr/local/bin/ && \
    rm terraform.zip

# Install Trivy.
# Previously this piped contrib/install.sh from the main branch into sh: an
# unpinned script off a mutable ref, executed as root at build time. Now a
# pinned release tarball is fetched and verified before the binary is used.
ARG TRIVY_VERSION=0.74.0
ARG TRIVY_SHA256=2ae6fe3ee734b7fdf11335663e18c75ea12dccc76062f09f164a3b0f8be4371a
RUN curl -fsSL --proto '=https' --proto-redir '=https' --tlsv1.2 -o trivy.tar.gz \
      "https://github.com/aquasecurity/trivy/releases/download/v${TRIVY_VERSION}/trivy_${TRIVY_VERSION}_Linux-64bit.tar.gz" && \
    echo "${TRIVY_SHA256}  trivy.tar.gz" | sha256sum -c - && \
    tar -xzf trivy.tar.gz trivy && \
    mv trivy /usr/local/bin/ && \
    rm trivy.tar.gz

# Copy Go binary
COPY --from=builder /app/smurf /usr/local/bin/smurf

# Copy entrypoint
COPY entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh /usr/local/bin/smurf

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]