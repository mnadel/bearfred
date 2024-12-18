GOBUILD_TAGS := fts5

clean:
	rm -rf target
	rm -rf package

init:
	mkdir -p target
	mkdir -p package

local:
	go build -tags $(GOBUILD_TAGS)

build: init
	GOOS=darwin GOARCH=amd64 go build -tags $(GOBUILD_TAGS) -o target/amd64/freddiebear
	GOOS=darwin GOARCH=arm64 go build -tags $(GOBUILD_TAGS) -o target/arm64/freddiebear

workflow:
	$(eval WFVER := $(shell git for-each-ref --sort=creatordate --format '%(refname)' refs/tags | tail -1 | cut -d/ -f3))
	grep -q $(WFVER) info.plist || (echo "Update version in info.plist to $(WFVER)"; exit 1)
	zip -r package/Freddiebear.alfredworkflow info.plist icon.png download.sh

package: build workflow
	gzip -c target/amd64/freddiebear > package/freddiebear.x86_64.gz
	gzip -c target/arm64/freddiebear > package/freddiebear.arm64.gz
