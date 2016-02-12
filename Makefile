PREFIX?=/usr/local
TAG=`git describe`

help:
	@echo "Targets:"
	@for x in $(.ALLTARGETS); do if [ "$$x" != ".END" ]; then printf "\t%s\n" $$x; fi; done
	@echo
	@echo "A fresh build might be: make patch-local-go-libucl test build"
	@echo

install: build
	install -o root -g wheel -m 755 bin/hfm ${PREFIX}/bin
	-mkdir -p ${PREFIX}/share/doc/hfm
	-mkdir -p ${PREFIX}/share/examples/hfm
	install -o root -g wheel -m 644 README.md ${PREFIX}/share/doc/hfm/
	install -o root -g wheel -m 644 doc/* ${PREFIX}/share/doc/hfm/
	install -o root -g wheel -m 644 examples/* ${PREFIX}/share/examples/hfm/
	install -o root -g wheel -m 644 examples/hfm.conf.sample ${PREFIX}/etc/

deinstall:
	-rm ${PREFIX}/bin/hfm
	-rm ${PREFIX}/share/doc/hfm/*
	-rm ${PREFIX}/share/examples/hfm/*
	-rm ${PREFIX}/etc/hfm.conf.sample
	-rmdir ${PREFIX}/share/doc/hfm
	-rmdir ${PREFIX}/share/examples/hfm

build: bin/hfm

clean:
	-rm -rf bin
	-rm -rf pkg
	-rm -rf vendor/src/github.com

test: deps
	gb test all -v

patch-local-go-libucl: vendor/src/github.com/mitchellh
	git apply --check vendor/patches/github.com/mitchellh/go-libucl/libucl.go.patch
	git apply vendor/patches/github.com/mitchellh/go-libucl/libucl.go.patch

bin/hfm: deps src/cmd/hfm/*.go
	gb build -ldflags "-X main.build_tag=${TAG} -X main.build_prefix=${PREFIX}" all

deps: vendor/src/github.com/mitchellh vendor/src/github.com/op

vendor/src/github.com/mitchellh:
	gb vendor restore

vendor/src/github.com/op:
	gb vendor restore
