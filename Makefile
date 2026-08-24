VERSION ?= 1.0.5
CHANNEL ?= test
NATIVE_SDK_PATH ?= $(HOME)/code/native
NATIVE ?= npx --yes @native-sdk/cli@0.7.0

.PHONY: test verify-sdk check-native generate-icons frontend-install frontend-lint frontend-test frontend-build frontend-verify native-helper dev-native build-native build-legacy package-native package-legacy verify-package

test:
	go test ./...
	go vet ./...

verify-sdk:
	NATIVE_SDK_PATH="$(NATIVE_SDK_PATH)" ./scripts/verify-native-sdk-pin.sh

check-native: verify-sdk frontend-build
	$(NATIVE) validate app.zon
	$(NATIVE) check
	PATH="$$HOME/.native/toolchains/zig-0.16.0:$$PATH" $(NATIVE) doctor --manifest app.zon --strict

generate-icons:
	./scripts/generate-macos-icons.sh assets
	./scripts/generate-provider-pngs.sh assets

frontend-install:
	cd frontend && pnpm install --frozen-lockfile

frontend-lint:
	cd frontend && pnpm lint

frontend-test:
	cd frontend && pnpm test

frontend-build:
	./scripts/build-frontend.sh

frontend-verify:
	node scripts/verify-frontend-dist.mjs frontend/dist

# Places the Go helper where unpackaged `native dev` hosts look first
# (zig-out/bin/../Resources/harnezpad). Packaged builds still embed via build-darwin.sh.
native-helper:
	mkdir -p zig-out/Resources
	@if [ -f assets/HarnezPadNative.icns ]; then \
		go build -o zig-out/Resources/harnezpad ./cmd/harnezpad; \
	elif [ -x dist/HarnezPad.app/Contents/Resources/harnezpad ]; then \
		cp dist/HarnezPad.app/Contents/Resources/harnezpad zig-out/Resources/harnezpad; \
	else \
		echo "native-helper: run make generate-icons, or make package-native first"; \
		exit 1; \
	fi

dev-native: native-helper
	$(NATIVE) dev

build-native: frontend-build
	$(NATIVE) build . --yes -Doptimize=ReleaseFast

build-legacy:
	go build ./cmd/harnezpad

package-native:
	./build-darwin.sh "$(VERSION)" "$(CHANNEL)" native

install-native:
	./scripts/install-macos-app.sh "$(CURDIR)/dist/HarnezPad.app"

package-legacy:
	./build-darwin.sh "$(VERSION)" "$(CHANNEL)" legacy

verify-package:
	./scripts/verify-macos-package.sh
