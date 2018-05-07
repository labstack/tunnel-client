IMAGE = labstack/tunnel
VERSION = 0.2.8

publish:
	git tag $(VERSION)
	git push origin --tags
	goreleaser --rm-dist
	docker build -t $(IMAGE):$(VERSION) -t $(IMAGE) .
	docker push $(IMAGE):$(VERSION)
	docker push $(IMAGE):latest

.PHONY: publish 
