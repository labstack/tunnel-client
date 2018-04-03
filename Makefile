VERSION = 0.1.2

build:
	git tag $(VERSION)
	git push origin --tags
	goreleaser --rm-dist

.PHONY: build
