FROM golang:1.23



WORKDIR /go/src/app
COPY . .

# Build the Go application
RUN go build -o smurf main.go
RUN chmod +x smurf


# Set entrypoint to the build binary
ENTRYPOINT ["./go/src/app/smurf"]