VERSION = 0.2.7

publish:
	git tag $(VERSION)
	git push origin --tags
	goreleaser --rm-dist

.PHONY: publish 
