FROM golang:1.23-alpine


WORKDIR /go/src/app
COPY . .

# Build the Go application
RUN go build -o smurf main.go && \
    mv smurf /usr/local/bin/ && \
    chmod +x /usr/local/bin/smurf

ENTRYPOINT ["/usr/local/bin/smurf"]