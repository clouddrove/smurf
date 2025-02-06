FROM golang:1.23-alpine

# Install required packages
RUN apk add --no-cache \
    git \
    curl \
    unzip \
    docker-cli

# Install Terraform
RUN curl -fsSL https://releases.hashicorp.com/terraform/1.5.7/terraform_1.5.7_linux_amd64.zip -o terraform.zip && \
    unzip terraform.zip && \
    mv terraform /usr/local/bin/ && \
    rm terraform.zip

# Set working directory
WORKDIR /go/src/app

# Copy application files
COPY . .

# Build the Go application
RUN go build -o smurf main.go && \
    mv smurf /usr/local/bin/ && \
    chmod +x /usr/local/bin/smurf

# Ensure the entrypoint.sh script is executable and has a shebang
RUN sed -i '1s|^|#!/bin/sh\n|' entrypoint.sh && \
    chmod +x entrypoint.sh && \
    mv entrypoint.sh /usr/local/bin/

# Set the entrypoint
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]








