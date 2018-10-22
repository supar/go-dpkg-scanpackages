.PHONY: build

UNAME_S=$(shell uname -s)

NAME = go-dpkg-scanpackages
VERSION = $(shell cat VERSION | sed -e 's,\-.*,,')
RELEASE = $(shell cat VERSION | sed -e 's,.*\-,,')


BUILD_DIR = $(notdir $(shell pwd))
BUILD_DATE = $(shell date +%Y%m%d%H%M%S)
BUILD_ARCH = amd64

ifeq (${UNAME_S}, Darwin)
BUILD_SYSTEM = darwin
else
BUILD_SYSTEM = linux
endif

hello-world:
	$(MAKE) BUILD_TARGET=hello-world \
		VERSION=1.0.0-1 \
		SOURCE="README.md"\
		BUILD_DEBIAN=fixtures/hello-world/debian \
		deb 
	cp $(shell find . -type f -path "*build/hello-world_1.0.0-1_amd64.deb" -print -quit) fixtures/

$(BUILD_TARGET)-$(VERSION): $(SOURCE)
	mkdir -p build/$@
	tar --transform "s,^,$@/src/$(BUILD_TARGET)/," -f build/$(BUILD_TARGET)_$(VERSION).orig.tar.gz -cz $^

$(BUILD_TARGET)-$(VERSION)/debian: $(BUILD_DEBIAN)
	cp -adR $^ build/$@

deb: .clean_deb $(BUILD_TARGET)-$(VERSION) $(BUILD_TARGET)-$(VERSION)/debian
	cd build/$(BUILD_TARGET)-$(VERSION) && debuild -us -uc -b

.clean_deb:
	@rm -rf $(shell find . -type d -path "*build/*" -print -quit)
