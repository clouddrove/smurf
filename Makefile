ifeq ($(shell go env GOOS),windows)
EXE=.exe
else
EXE=
endif

DIST=dist
BINDIR=.

BASENAME=$(notdir $(shell pwd))
PROGRAM=$(BINDIR)/$(BASENAME)$(EXE)
LAST_RELEASE=

REPO=$(shell go list | head -n 1)
IMAGE=$(BASENAME)
VERSION ?= $(shell git describe --tags --always --dirty)
DOCKER=docker
PACKAGE=$(DIST)/$(basename $(notdir $(PROGRAM)))-$(shell go env GOOS)-$(shell go env GOARCH).zip


.PHONY: $(PROGRAM)

all: $(PROGRAM)

compile: $(PROGRAM)

$(PROGRAM): $(BINDIR)
	mkdir -p $(dir $@)
	go build -ldflags="-X '$(REPO)/program.Version=${VERSION}'" -o $(PROGRAM)

package: $(PACKAGE)

$(PACKAGE): $(PROGRAM)

# These next 2 recipes know how to make .zip and .tar files, which are used implicitly in making the package
%.zip:
	mkdir $(dir $@)
	zip -j $@ $?

%.tar.gz %.tgz:
	mkdir $(dir $@)
	tar -czf $@ -C $(dir $<) $(notdir $<)

install:
	go install -ldflags="-X '$(REPO)/program.Version=${VERSION}'"

image:
	$(DOCKER) build -f Dockerfile --build-arg PROGRAM=$(BASENAME) --build-arg VERSION=$(VERSION) --build-arg BASENAME=$(BASENAME) -t $(IMAGE) .

test:
	go test -v ./...

vet:
	go vet ./...

changelog: CHANGELOG.md
CHANGELOG.md: .chglog/config.yml
	git chglog $(LAST_RELEASE) >$@

.chglog/config.yml: go.mod
	sed -i.bak -e "s|repository_url:.*|repository_url: https://$(REPO)|" $@

hooks: .git/hooks/pre-commit

.git/hooks/pre-commit: .pre-commit-config.yaml
	pre-commit install
	pre-commit install --hook-type commit-msg


info::
	@echo BASENAME=$(BASENAME)
	@echo PROGRAM=$(PROGRAM)
	@echo IMAGE=$(IMAGE)


tools:
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/go-critic/go-critic/cmd/gocritic@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	