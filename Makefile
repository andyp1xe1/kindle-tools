.PHONY: install lint fmt clean
.SECONDARY:

TOOLS   := wallpapers jsrepl
DESTDIR ?= /run/media/moss/Kindle

dev-%:
	go run ./cmd/$* -dev

build-%:
	@mkdir -p dist/$*
	GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 \
		go build -trimpath -ldflags="-s -w" -o dist/$*/$* ./cmd/$*
	@ls -lh dist/$*/$*

package-%: build-%
	cp scriptlets/$*/$*.sh dist/$*.sh
	chmod +x dist/$*.sh
	cp scriptlets/$*/config.xml scriptlets/$*/index.html scriptlets/$*/migrate.sql dist/$*/
	@if [ -d koreader/$*.koplugin ]; then \
		rm -rf dist/$*.koplugin; \
		cp -r koreader/$*.koplugin dist/$*.koplugin; \
	fi

install-%: package-%
	@test -d "$(DESTDIR)" || { echo "DESTDIR=$(DESTDIR) not found - plug in/unlock the Kindle"; exit 1; }
	rm -rf "$(DESTDIR)/$*"
	cp -r dist/$* "$(DESTDIR)/"
	cp dist/$*.sh "$(DESTDIR)/documents/"
	@if [ -d dist/$*.koplugin ]; then \
		rm -rf "$(DESTDIR)/koreader/plugins/$*.koplugin"; \
		cp -r dist/$*.koplugin "$(DESTDIR)/koreader/plugins/"; \
	fi

install: $(addprefix install-,$(TOOLS))
	sync
	@echo "Installed to $(DESTDIR) - safe to eject"

lint:
	@d=$$(gofmt -l cmd/ internal/); \
	  if [ -n "$$d" ]; then echo "needs gofmt:"; echo "$$d"; exit 1; fi
	go vet ./...
	shellcheck scriptlets/*/*.sh
	biome check internal/
	alejandra --check flake.nix

fmt:
	gofmt -w cmd/ internal/
	biome check --write internal/
	alejandra flake.nix

clean:
	rm -rf dist
