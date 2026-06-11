.PHONY: swagger license license-check build-embedded build-test cross-build

swagger:
	scripts/swagger.sh

license:
	scripts/update_go_license.sh

license-check:
	scripts/update_go_license.sh --check

build-embedded:
	cd frontend && pnpm build:embed
	rm -rf internal/router/dist
	cp -R frontend/out internal/router/dist
	go build -tags embed_frontend -o bin/pixez-sync main.go

code-check:
	golangci-lint run
	cd frontend && pnpm tsc --noEmit --jsx preserve && npx eslint . --max-warnings 0

build-test:
	@echo "==> Running frontend and backend build tests in parallel..."
	@PIDS=""; \
	STATUS=0; \
	( cd frontend && pnpm build 2>&1 | sed 's/^/[frontend] /' ) & PIDS="$$PIDS $$!"; \
	( go test ./... && go build -o /dev/null ./... 2>&1 | sed 's/^/[backend]  /' ) & PIDS="$$PIDS $$!"; \
	for PID in $$PIDS; do \
		wait $$PID || STATUS=1; \
	done; \
	if [ $$STATUS -eq 0 ]; then \
		echo "==> All build tests passed."; \
	else \
		echo "==> Build test FAILED." >&2; \
		exit 1; \
	fi

cross-build:
	@echo "==> Cross-compiling \
	$(if $(GOOS),$(GOOS),linux/darwin/windows) × \
	$(if $(GOARCH),$(GOARCH),amd64/arm64) \
	(version=$(or $(VERSION),dev))..."
	@mkdir -p bin
	docker build \
		--file docker/Dockerfile.cross \
		--target export \
		--build-arg VERSION=$(or $(VERSION),dev) \
		--build-arg BUILD_DATE="$(shell date -u +'%Y-%m-%d %H:%M:%S')" \
		$(if $(GOOS),--build-arg TARGET_OS=$(GOOS)) \
		$(if $(GOARCH),--build-arg TARGET_ARCH=$(GOARCH)) \
		--output type=local,dest=./bin \
		.
	@echo "==> Done. Binaries written to ./bin/"
	@ls -lh bin/
