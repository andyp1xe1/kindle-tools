.PHONY: dev wall-build repl-build wall-package repl-package install lint fmt clean

NAME     := wallpapers
PKG      := ./cmd/$(NAME)
BIN      := dist/$(NAME)/$(NAME)
REPL     := jsrepl
REPL_PKG := ./cmd/$(REPL)
REPL_BIN := dist/$(REPL)/$(REPL)
DESTDIR  ?= /run/media/moss/Kindle

dev:
	go run $(PKG) -dev

wall-build:
	@mkdir -p $(dir $(BIN))
	GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 \
		go build -trimpath -ldflags="-s -w" -o $(BIN) $(PKG)
	@ls -lh $(BIN)

repl-build:
	@mkdir -p $(dir $(REPL_BIN))
	GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 \
		go build -trimpath -ldflags="-s -w" -o $(REPL_BIN) $(REPL_PKG)
	@ls -lh $(REPL_BIN)

wall-package: wall-build
	cp scriptlet/$(NAME).sh dist/$(NAME).sh
	chmod +x dist/$(NAME).sh
	cp scriptlet/config.xml scriptlet/index.html scriptlet/migrate.sql dist/$(NAME)/
	rm -rf dist/$(NAME).koplugin
	cp -r koreader/$(NAME).koplugin dist/$(NAME).koplugin

repl-package: repl-build
	cp $(REPL)/$(REPL).sh dist/$(REPL).sh
	chmod +x dist/$(REPL).sh
	cp $(REPL)/config.xml $(REPL)/index.html $(REPL)/migrate.sql dist/$(REPL)/

install: wall-package repl-package
	@test -d "$(DESTDIR)" || { echo "DESTDIR=$(DESTDIR) not found - plug in/unlock the Kindle"; exit 1; }
	rm -rf "$(DESTDIR)/$(NAME)"
	cp -r dist/$(NAME) "$(DESTDIR)/"
	cp dist/$(NAME).sh "$(DESTDIR)/documents/"
	rm -rf "$(DESTDIR)/koreader/plugins/$(NAME).koplugin"
	cp -r dist/$(NAME).koplugin "$(DESTDIR)/koreader/plugins/"
	rm -rf "$(DESTDIR)/$(REPL)"
	cp -r dist/$(REPL) "$(DESTDIR)/"
	cp dist/$(REPL).sh "$(DESTDIR)/documents/"
	sync
	@echo "Installed to $(DESTDIR) - safe to eject"

lint:
	@d=$$(gofmt -l cmd/ internal/); \
	  if [ -n "$$d" ]; then echo "needs gofmt:"; echo "$$d"; exit 1; fi
	go vet ./...
	shellcheck scriptlet/*.sh
	biome check internal/web/
	alejandra --check flake.nix

fmt:
	gofmt -w cmd/ internal/
	biome check --write internal/web/
	alejandra flake.nix

clean:
	rm -rf dist dev
