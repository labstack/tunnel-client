VERSION = 0.1.1

clean:
	rm -rf build

build: clean
	xgo --targets=darwin-10.8/amd64,linux/amd64,linux/arm-6,linux/arm-7,linux/arm64,windows-8.0/amd64 --pkg cmd/tunnel -out build/tunnel-$(VERSION) github.com/labstack/tunnel

.PHONY: clean build
