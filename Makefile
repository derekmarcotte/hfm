ETCDIR?=examples
TAG?=`git describe --tags`

help:
	@echo "Targets:"
	@for x in $(.ALLTARGETS); do if [ "$$x" != ".END" ]; then printf "\t%s\n" $$x; fi; done
	@echo
	@echo "A fresh build might be: make test build"
	@echo

build: bin/hfm

clean:
	-rm -rf bin

test:
	go test -short ./...

test-long:
	go test ./...

bin/hfm: cmd/hfm/*.go go.mod go.sum
	CGO_ENABLED=0 go build -o bin/hfm -ldflags "-X main.build_tag=${TAG} -X main.build_etcdir=${ETCDIR}" ./cmd/hfm
