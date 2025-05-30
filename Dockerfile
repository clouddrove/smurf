# Stage 1: Build Go application
FROM golang:1.24-alpine as builder

RUN apk add --no-cache git
WORKDIR /app
COPY . .
RUN go build -o smurf main.go

# Stage 2: Create minimal runtime image
FROM alpine:3.18

# Install only essential CLI tools
RUN apk add --no-cache \
    bash \
    curl \
    unzip \
    git \
    docker-cli \
    aws-cli

# Install Google Cloud SDK (minimal install)
ENV CLOUDSDK_INSTALL_DIR /usr/local/gcloud
RUN curl -sSL https://sdk.cloud.google.com | bash -s -- --disable-prompts --install-dir=${CLOUDSDK_INSTALL_DIR} && \
    rm -rf ${CLOUDSDK_INSTALL_DIR}/.install/.backup

ENV PATH $PATH:${CLOUDSDK_INSTALL_DIR}/google-cloud-sdk/bin

# Install GKE plugin
RUN gcloud components install gke-gcloud-auth-plugin --quiet

# Install Terraform
RUN curl -fsSL https://releases.hashicorp.com/terraform/1.5.7/terraform_1.5.7_linux_amd64.zip -o terraform.zip && \
    unzip terraform.zip && \
    mv terraform /usr/local/bin/ && \
    rm terraform.zip

# Install Trivy
RUN curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin

# Copy Go binary
COPY --from=builder /app/smurf /usr/local/bin/smurf

# Copy entrypoint
COPY entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh /usr/local/bin/smurf

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]