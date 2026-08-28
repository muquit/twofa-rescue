# Aug-03-2026 
BINARY  := twofa-rescue
VERSION := $(shell cat VERSION)
LDFLAGS := -ldflags "-w -s -X main.version=$(VERSION)"
BUILD_OPTIONS := -trimpath

.PHONY: all build clean docs doc gen_usage test clean_exif

all: build

build_native:
	go build $(BUILD_OPTIONS) $(LDFLAGS) -o $(BINARY) .

test:
	go test ./...

build: clean
	@echo ">>>> Compiling native binary ..."
	go build $(BUILD_OPTIONS) $(LDFLAGS) -o $(BINARY) .
	@echo ""
	@echo ">>>> Compiling cross-platform binaries ..."
	go-xbuild-go \
		-pi=false \
		-build-args '$(BUILD_OPTIONS) $(LDFLAGS)'

gen_usage:
	./scripts/gen_usage.sh

docs: build_native gen_usage
	markdown-toc-go -i docs/README.md \
        -o ./README.md --glossary docs/glossary.txt -f
	markdown-toc-go -i docs/ChangeLog.md \
		-o ./ChangeLog.md --glossary docs/glossary.txt -f -no-credit

doc: docs

# check if GITHUB_TOKEN is set and valid, fail the build otherwise
check_github_token:
	@if [ -z "$(GITHUB_TOKEN)" ]; then \
        echo "*** ERROR: GITHUB_TOKEN is not set"; \
        exit 1; \
    fi
	@status=$$(curl -s -o /tmp/check_github_token.$$$$.json -w '%{http_code}' \
        -H "Authorization: token $(GITHUB_TOKEN)" https://api.github.com/user); \
    if [ "$$status" != "200" ]; then \
        echo "*** ERROR: GITHUB_TOKEN is not valid (HTTP $$status)"; \
        cat /tmp/check_github_token.$$$$.json; \
        rm -f /tmp/check_github_token.$$$$.json; \
        exit 1; \
    fi; \
    jq '{login, name, type}' < /tmp/check_github_token.$$$$.json; \
    rm -f /tmp/check_github_token.$$$$.json
	@curl -sI -H "Authorization: token $(GITHUB_TOKEN)" \
        https://api.github.com/user | grep -i x-oauth-scopes

mk_release_notes:
	./scripts/mk_release_notes.sh

clean_exif:
	@echo ">>>> Removing EXIF metadata from images ..."
	exiftool -all= -overwrite_original images/*.png
	rm -f images/*_original

release: check_github_token clean_exif mk_release_notes
	@echo "*** Releasing on github ..."
	go-xbuild-go -release


clean:
	rm -rf ./bin
	rm -rf ./Formula
	rm -f $(BINARY)
