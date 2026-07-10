.PHONY: help default build test lint vulncheck clean install bump-version release release-prepare release-clean release-build release-checksums release-sbom release-formula homebrew release-publish release-commit

VERSION_FILE  := cmd/rename/VERSION
VERSION       := $(shell cat $(VERSION_FILE) 2>/dev/null | tr -d '[:space:]')
# Release tag in the shared tap repo is project-prefixed (rename-vX.Y.Z) so it
# never collides with sibling projects that publish to the same repo.
RELEASE_TAG   := rename-v$(VERSION)
GITHUB_REPO   := simonski/rename
# Release binaries are hosted on the PUBLIC tap repo so `brew install` can
# download them anonymously even when the source repo ($(GITHUB_REPO)) is
# private. The formula urls in homebrew/rename.rb.tmpl point at this repo.
DIST_REPO     := simonski/homebrew-tap
TAP_REPO      := simonski/homebrew-tap
DIST_DIR      := dist
RELEASE_DARWIN_ARM64 := $(DIST_DIR)/rename_$(VERSION)_darwin_arm64.tar.gz
RELEASE_DARWIN_AMD64 := $(DIST_DIR)/rename_$(VERSION)_darwin_amd64.tar.gz
RELEASE_LINUX_AMD64  := $(DIST_DIR)/rename_$(VERSION)_linux_amd64.tar.gz
RELEASE_LINUX_ARM64  := $(DIST_DIR)/rename_$(VERSION)_linux_arm64.tar.gz
RELEASE_TARBALLS := $(RELEASE_DARWIN_ARM64) $(RELEASE_DARWIN_AMD64) $(RELEASE_LINUX_AMD64) $(RELEASE_LINUX_ARM64)

default: help

help:
	@printf "Available targets:\n\n"
	@printf "Build & test:\n"
	@printf "  make build            Build ./bin/rename (does not change the version).\n"
	@printf "  make test             Run all Go tests.\n"
	@printf "  make lint             golangci-lint + gosec.\n"
	@printf "  make vulncheck        govulncheck on all Go packages.\n"
	@printf "  make install          Build and copy to GOPATH/bin.\n"
	@printf "  make clean            Remove built binaries from ./bin.\n\n"
	@printf "Release:\n"
	@printf "  make release          Full release: bump + binaries + checksums + SBOM + formula + tap.\n"
	@printf "  make homebrew         Regenerate the formula and push it to the tap.\n\n"

# build compiles the binary WITHOUT touching the version. The version only
# advances when releasing (see release), so day-to-day builds never churn
# cmd/rename/VERSION.
build:
	@mkdir -p bin
	go build -o ./bin/rename ./cmd/rename

test:
	go test ./...

lint:
	golangci-lint run ./...
	gosec ./...

vulncheck:
	govulncheck ./...

install: build
	cp ./bin/rename $$(go env GOPATH)/bin/rename

clean:
	@rm -rf bin

bump-version:
	@if [ ! -f "$(VERSION_FILE)" ]; then \
		printf "0.1.0\n" > "$(VERSION_FILE)"; \
	else \
		version=$$(tr -d '[:space:]' < "$(VERSION_FILE)"); \
		major=$${version%%.*}; \
		rest=$${version#*.}; \
		minor=$${rest%%.*}; \
		patch=$${rest##*.}; \
		patch=$$((patch + 1)); \
		printf "%s.%s.%s\n" "$$major" "$$minor" "$$patch" > "$(VERSION_FILE)"; \
	fi

release-prepare:
	@$(MAKE) bump-version

# ─── release ──────────────────────────────────────────────────────────────────
# Produces cross-platform tarballs in ./dist, creates a GitHub release, and
# pushes the updated Homebrew formula to the simonski/homebrew-tap repo.
# Prerequisites: go, gh (GitHub CLI), shasum, cyclonedx-gomod, git.

RELEASE_PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64

release-clean:
	@rm -rf $(DIST_DIR)

release-build:
	@$(MAKE) release-clean
	@mkdir -p $(DIST_DIR)
	@echo "Building v$(VERSION) for all platforms..."
	@for platform in $(RELEASE_PLATFORMS); do \
		os=$$(echo $$platform | cut -d/ -f1); \
		arch=$$(echo $$platform | cut -d/ -f2); \
		name=rename_$(VERSION)_$${os}_$${arch}; \
		outdir=$(DIST_DIR)/$$name; \
		mkdir -p $$outdir; \
		printf "  %-32s" "$$os/$$arch"; \
		GOOS=$$os GOARCH=$$arch go build -o $$outdir/rename ./cmd/rename && echo "ok" || exit 1; \
		tar -czf $(DIST_DIR)/$${name}.tar.gz -C $$outdir rename; \
		rm -rf $$outdir; \
	done
	@echo "Tarballs written to $(DIST_DIR)/"

release-checksums: release-build
	@echo "Computing SHA256 checksums..."
	@cd $(DIST_DIR) && \
		for f in *.tar.gz; do \
			shasum -a 256 "$$f"; \
		done | tee checksums.txt
	@echo "Checksums written to $(DIST_DIR)/checksums.txt"

release-sbom:
	@echo "Generating CycloneDX SBOM..."
	@cyclonedx-gomod mod -json -output $(DIST_DIR)/sbom.cdx.json .
	@echo "SBOM written to $(DIST_DIR)/sbom.cdx.json"

# NB: no build dependency on purpose — this reads the already-built artifacts in
# $(DIST_DIR) so the formula checksums match the tarballs that were uploaded to
# the release. Run release-checksums (or release) first. Rebuilding here would
# regenerate the tarballs (tar/gzip embed mtimes) and produce mismatched sha256s.
release-formula:
	@echo "Generating homebrew/rename.rb for v$(VERSION)..."
	@for f in $(RELEASE_TARBALLS); do \
		if [ ! -f "$$f" ]; then \
			echo "Missing release artifact: $$f"; \
			exit 1; \
		fi; \
	done
	@darwin_arm64=$$(awk '/ rename_$(VERSION)_darwin_arm64.tar.gz$$/{print $$1}' $(DIST_DIR)/checksums.txt); \
	 darwin_amd64=$$(awk '/ rename_$(VERSION)_darwin_amd64.tar.gz$$/{print $$1}' $(DIST_DIR)/checksums.txt); \
	 linux_amd64=$$(awk '/ rename_$(VERSION)_linux_amd64.tar.gz$$/{print $$1}' $(DIST_DIR)/checksums.txt); \
	 linux_arm64=$$(awk '/ rename_$(VERSION)_linux_arm64.tar.gz$$/{print $$1}' $(DIST_DIR)/checksums.txt); \
	 if [ -z "$$darwin_arm64" ] || [ -z "$$darwin_amd64" ] || [ -z "$$linux_amd64" ] || [ -z "$$linux_arm64" ]; then \
		echo "Missing release checksums in $(DIST_DIR)/checksums.txt"; \
		exit 1; \
	 fi; \
	 sed \
		-e "s/__VERSION__/$(VERSION)/g" \
		-e "s/__DARWIN_ARM64_SHA256__/$$darwin_arm64/g" \
		-e "s/__DARWIN_AMD64_SHA256__/$$darwin_amd64/g" \
		-e "s/__LINUX_AMD64_SHA256__/$$linux_amd64/g" \
		-e "s/__LINUX_ARM64_SHA256__/$$linux_arm64/g" \
		 homebrew/rename.rb.tmpl > homebrew/rename.rb
	@echo "Formula written to homebrew/rename.rb"

homebrew: release-formula
	@echo "Updating homebrew tap..."
	@TAP_DIR=$$(mktemp -d) && \
		trap "rm -rf $$TAP_DIR" EXIT && \
		gh repo clone $(TAP_REPO) "$$TAP_DIR" && \
		cp homebrew/rename.rb "$$TAP_DIR/Formula/rename.rb" && \
		if [ -z "$$(git -C "$$TAP_DIR" status --porcelain -- Formula/rename.rb)" ]; then \
			echo "Homebrew tap already up to date."; \
			exit 0; \
		fi && \
		git -C "$$TAP_DIR" add Formula/rename.rb && \
		git -C "$$TAP_DIR" commit -m "rename $(VERSION)" && \
		git -C "$$TAP_DIR" push
	@echo "Homebrew tap updated."

release-publish: release-build release-checksums release-sbom
	@if gh release view $(RELEASE_TAG) --repo $(DIST_REPO) >/dev/null 2>&1; then \
		echo "Release $(RELEASE_TAG) already exists; aborting."; \
		exit 1; \
	fi
	@echo "Creating GitHub release $(RELEASE_TAG) on $(DIST_REPO)..."
	@gh release create $(RELEASE_TAG) \
		--repo $(DIST_REPO) \
		--title "rename v$(VERSION)" \
		--notes "Release v$(VERSION)" \
		$(DIST_DIR)/rename_$(VERSION)_darwin_arm64.tar.gz \
		$(DIST_DIR)/rename_$(VERSION)_darwin_amd64.tar.gz \
		$(DIST_DIR)/rename_$(VERSION)_linux_amd64.tar.gz \
		$(DIST_DIR)/rename_$(VERSION)_linux_arm64.tar.gz \
		$(DIST_DIR)/checksums.txt \
		$(DIST_DIR)/sbom.cdx.json
	@echo "Release $(RELEASE_TAG) published."
	@$(MAKE) homebrew
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "  v$(VERSION) released"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "  Install with:"
	@echo "    brew install simonski/tap/rename"
	@echo ""

release-commit:
	@echo "Committing release $(RELEASE_TAG)..."
	@git add $(VERSION_FILE) homebrew/rename.rb
	@if git diff --cached --quiet; then \
		echo "No release files changed; nothing to commit."; \
	else \
		git commit -m "Release $(RELEASE_TAG)" && \
		git push; \
	fi

# Full end-to-end release. Each step runs as a recursive sub-make so that
# $(VERSION) (expanded at parse time) is re-read from cmd/rename/VERSION AFTER
# the bump — otherwise publish would reuse the old, already-released version.
release:
	@$(MAKE) release-prepare
	@$(MAKE) release-publish
	@$(MAKE) release-commit
