help:
	@echo "Targets:"
	@for x in $(.ALLTARGETS); do if [ "$$x" != ".END" ]; then printf "\t%s\n" $$x; fi; done

build: 
	gb build all

test: 
	gb test all -v

deps:
	gb vendor restore

patch-local-go-libucl:
	git apply --check vendor/patches/github.com/mitchellh/go-libucl/libucl.go.patch
	git apply vendor/patches/github.com/mitchellh/go-libucl/libucl.go.patch

