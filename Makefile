ETCDIR?=examples
TAG?=`git describe`

help:
	@echo "Targets:"
	@for x in $(.ALLTARGETS); do if [ "$$x" != ".END" ]; then printf "\t%s\n" $$x; fi; done
	@echo
	@echo "A fresh build might be: make patch-local-go-libucl test build"
	@echo

build: bin/hfm

clean:
	-rm -rf bin
	-rm -rf pkg
	-rm -rf vendor/src/github.com

test: deps
	gb test all -v

patch-local-go-libucl: deps
	git apply --check vendor/patches/github.com/mitchellh/go-libucl/libucl.go.patch
	git apply vendor/patches/github.com/mitchellh/go-libucl/libucl.go.patch

bin/hfm: deps src/cmd/hfm/*.go
	gb build -ldflags "-X main.build_tag=${TAG} -X main.build_etcdir=${ETCDIR} -extldflags '-static'" all

deps: vendor/src/github.com/mitchellh vendor/src/github.com/op

vendor/src/github.com/mitchellh:
	gb vendor restore

vendor/src/github.com/op:
	gb vendor restore
