VERSION = 0.2.4

publish:
	git tag $(VERSION)
	git push origin --tags
	goreleaser --rm-dist

.PHONY: publish 
