

help:
	@echo $(.ALLTARGETS)

build: 
	gb build all

test: 
	gb test all -v

deps:
	gb vendor restore

patch-local-go-libucl:
	git apply --check vendor/patches/github.com/mitchellh/go-libucl/libucl.go.patch
	git apply vendor/patches/github.com/mitchellh/go-libucl/libucl.go.patch

