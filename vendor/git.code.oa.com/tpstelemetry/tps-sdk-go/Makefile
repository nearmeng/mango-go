ALL_SRC := $(shell find . -name '*.go' \
							-not -path './opentelemetry/proto/*' \
							-type f | sort)
ALL_DOC := $(shell find . \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)

ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)

ADDLICENCESE=addlicense
MISSPELL=misspell -error
IMPI=impi

.PHONY: precommit
precommit: fmt lint build checklicense examples

.PHONY: impi
impi:
	@$(IMPI) --local git.code.oa.com/tpstelemetry/tps-sdk-go --scheme stdThirdPartyLocal --skip example/trpc/protocol --skip opentelemetry/proto ./...

.PHONY: checklicense
checklicense:
	@ADDLICENCESEOUT=`$(ADDLICENCESE) -check $(ALL_SRC) 2>&1`; \
		if [ "$$ADDLICENCESEOUT" ]; then \
			echo "$(ADDLICENCESE) FAILED => add License errors:\n"; \
			echo "$$ADDLICENCESEOUT\n"; \
			echo "Use 'make addlicense' to fix this."; \
			exit 1; \
		else \
			echo "Check License finished successfully"; \
		fi

.PHONY: addlicense
addlicense:
	$(ADDLICENCESE) -c 'The TpsTelemetry Authors' $(ALL_SRC)

.PHONY: misspell
misspell:
	$(MISSPELL) $(ALL_DOC)

.PHONY: lint
lint:
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  echo "go mod tidy in $${dir}"; \
	  (cd "$${dir}" && \
	    go mod tidy); \
	done
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  echo "go mod tidy in $${dir}"; \
	  (cd "$${dir}" && \
	    go mod tidy); \
	done
	set -e; for dir in $(ALL_GO_MOD_DIRS); do \
    	  echo "golangci-lint in $${dir}"; \
    	  (cd "$${dir}" && \
    	    golangci-lint run --fix && \
    	    golangci-lint run); \
    done

.PHONY: build
build:
	@set -e; for dir in $(ALL_GO_MOD_DIRS); do \
	  (cd "$${dir}" && \
	    go build -v ./...; \
		echo "Build $${dir} success!"; \
	); \
	done
.PHONY: examples
examples:
	@cd example/basic; \
	go build -v 
	@echo "Build basic example success!"

	@cd example/log; \
	go build -v 
	@echo "Build log example success!"

	@cd example/stdout/remote; \
    go build -v
	@echo "Build stdout/remote example success!"

	@cd example/stdout/std; \
    go build -v
	@echo "Build stdout/std example success!"

.PHONY: fmt
fmt:
	gofmt -w -s .
	goimports -w -local git.code.oa.com/tpstelemetry/tps-sdk-go ./

