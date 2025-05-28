# Stage 1: Build Go application
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy dependency files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy remaining files and build
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o smurf .

# Stage 2: Minimal runtime image
FROM alpine:3.18

# Install base tools (adjust as needed)
RUN apk add --no-cache \
    bash \
    ca-certificates \
    curl \
    git \
    docker-cli \
    aws-cli \
    unzip

# Install optional tools via build arguments
ARG INSTALL_AWS_CLI=false
ARG INSTALL_GCLOUD=false
ARG INSTALL_TERRAFORM=false

# Conditionally install AWS CLI
RUN if [ "$INSTALL_AWS_CLI" = "true" ]; then \
      apk add --no-cache aws-cli; \
    fi

# Conditionally install Google Cloud SDK
RUN if [ "$INSTALL_GCLOUD" = "true" ]; then \
      apk add --no-cache python3 && \
      curl -sSL https://sdk.cloud.google.com | bash -s -- --disable-prompts --install-dir=/usr/local/gcloud && \
      ln -s /usr/local/gcloud/google-cloud-sdk/bin/gcloud /usr/local/bin/gcloud && \
      ln -s /usr/local/gcloud/google-cloud-sdk/bin/gsutil /usr/local/bin/gsutil; \
    fi

# Conditionally install Terraform
RUN if [ "$INSTALL_TERRAFORM" = "true" ]; then \
      curl -fsSL https://releases.hashicorp.com/terraform/1.5.7/terraform_1.5.7_linux_amd64.zip -o terraform.zip && \
      unzip terraform.zip && \
      mv terraform /usr/local/bin/ && \
      rm terraform.zip; \
    fi

# Copy compiled binary and entrypoint
COPY --from=builder /app/smurf /usr/local/bin/
COPY entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh /usr/local/bin/smurf

# Proper entrypoint configuration
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]