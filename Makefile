.PHONY: test ctest covdir bindir coverage docs linter qtest clean dep release logo license
PLUGIN_NAME="caddy-git"
PLUGIN_VERSION:=$(shell cat VERSION | head -1)
GIT_COMMIT:=$(shell git describe --dirty --always)
GIT_BRANCH:=$(shell git rev-parse --abbrev-ref HEAD -- | head -1)
LATEST_GIT_COMMIT:=$(shell git log --format="%H" -n 1 | head -1)
BUILD_USER:=$(shell whoami)
BUILD_DATE:=$(shell date +"%Y-%m-%d")
BUILD_DIR:=$(shell pwd)
CADDY_VERSION="v2.5.0"

all: info 
	@mkdir -p bin/
	@rm -rf ./bin/caddy
	@rm -rf ../xcaddy-$(PLUGIN_NAME)/*
	@mkdir -p ../xcaddy-$(PLUGIN_NAME) && cd ../xcaddy-$(PLUGIN_NAME) && \
		xcaddy build $(CADDY_VERSION) --output ../$(PLUGIN_NAME)/bin/caddy \
		--with github.com/greenpau/caddy-git@$(LATEST_GIT_COMMIT)=$(BUILD_DIR)
	@#--with github.com/go-git/go-git/v5@v5.3.0=/home/greenpau/dev/go/src/github.com/go-git/go-git
	@#bin/caddy run -config assets/config/Caddyfile
	@for f in `find ./assets -type f -name 'Caddyfile'`; do bin/caddy fmt -overwrite $$f; done

info:
	@echo "DEBUG: Version: $(PLUGIN_VERSION), Branch: $(GIT_BRANCH), Revision: $(GIT_COMMIT)"
	@echo "DEBUG: Build on $(BUILD_DATE) by $(BUILD_USER)"

linter:
	@echo "DEBUG: running lint checks"
	@golint -set_exit_status ./...
	@echo "DEBUG: completed $@"

test: covdir linter
	@echo "DEBUG: running tests"
	@go test -v -coverprofile=.coverage/coverage.out ./...
	@echo "DEBUG: completed $@"

ctest: covdir linter
	@echo "DEBUG: running tests"
	@time richgo test -v -coverprofile=.coverage/coverage.out ./...
	@echo "DEBUG: completed $@"

covdir:
	@echo "DEBUG: creating .coverage/ directory"
	@mkdir -p .coverage
	@echo "DEBUG: completed $@"

bindir:
	@echo "DEBUG: creating bin/ directory"
	@mkdir -p bin/
	@echo "DEBUG: completed $@"

coverage: covdir
	@echo "DEBUG: running coverage"
	@go tool cover -html=.coverage/coverage.out -o .coverage/coverage.html
	@go test -covermode=count -coverprofile=.coverage/coverage.out ./...
	@go tool cover -func=.coverage/coverage.out | grep -v "100.0"
	@echo "DEBUG: completed $@"

clean:
	@rm -rf .coverage/
	@rm -rf bin/
	@echo "DEBUG: completed $@"

qtest: covdir
	@echo "DEBUG: perform quick tests ..."
	@#go test -v -coverprofile=.coverage/coverage.out -run TestApp ./*.go
	@#go test -v -coverprofile=.coverage/coverage.out -run TestParseCaddyfile ./*.go
	@time richgo test -v -coverprofile=.coverage/coverage.out -run TestParseCaddyfileAppConfig ./*.go
	@#go test -v -coverprofile=.coverage/coverage.out -run Test* ./pkg/services/...
	@go tool cover -html=.coverage/coverage.out -o .coverage/coverage.html
	@go tool cover -func=.coverage/coverage.out | grep -v "100.0"
	@echo "DEBUG: completed $@"

dep:
	@echo "Making dependencies check ..."
	@go install golang.org/x/lint/golint@latest
	@go install github.com/caddyserver/xcaddy/cmd/xcaddy@latest
	@go install github.com/greenpau/versioned/cmd/versioned@latest
	@go install github.com/kyoh86/richgo@latest

release:
	@echo "Making release"
	@go mod tidy
	@go mod verify
	@if [ $(GIT_BRANCH) != "main" ]; then echo "cannot release to non-main branch $(GIT_BRANCH)" && false; fi
	@git diff-index --quiet HEAD -- || ( echo "git directory is dirty, commit changes first" && false )
	@versioned -patch
	@echo "Patched version"
	@git add VERSION
	@git commit -m "released v`cat VERSION | head -1`"
	@git tag -a v`cat VERSION | head -1` -m "v`cat VERSION | head -1`"
	@git push
	@git push --tags
	@@echo "If necessary, run the following commands:"
	@echo "  git push --delete origin v$(PLUGIN_VERSION)"
	@echo "  git tag --delete v$(PLUGIN_VERSION)"

logo:
	@mkdir -p assets/docs/images
	@gm convert -background black -font Bookman-Demi \
		-size 640x320 "xc:black" \
		-pointsize 72 \
		-draw "fill white gravity center text 0,0 'caddy\ngit'" \
		assets/docs/images/logo.png

license:
	@for f in `find ./ -type f -name '*.go'`; do versioned -addlicense -copyright="Paul Greenberg greenpau@outlook.com" -year=2022 -filepath=$$f; done
