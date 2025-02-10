FROM golang:1.23-alpine

# Install required packages
RUN apk add --no-cache \
    git \
    curl \
    unzip \
    docker-cli \
    aws-cli \
    bash

# Install Terraform
RUN curl -fsSL https://releases.hashicorp.com/terraform/1.5.7/terraform_1.5.7_linux_amd64.zip -o terraform.zip && \
    unzip terraform.zip && \
    mv terraform /usr/local/bin/ && \
    rm terraform.zip

# Set working directory
WORKDIR /go/src/app

# Copy application files
COPY . .

# Ensure entrypoint.sh is copied and executable
COPY entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh

# Build the Go application
RUN go build -o smurf main.go && \
    mv smurf /usr/local/bin/ && \
    chmod +x /usr/local/bin/smurf

# Set the entrypoint
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]