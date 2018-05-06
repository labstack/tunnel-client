VERSION = 0.2.8

publish:
	git tag $(VERSION)
	git push origin --tags
	goreleaser --rm-dist

.PHONY: publish 
