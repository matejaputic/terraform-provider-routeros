PYTHON ?= uv run --locked python

default: test build

build:
	go build -v ./...

install: build
	go install -v ./...

lint: lintpython
	golangci-lint run

lintpython:
	uv run --locked ruff check tools
	uv run --locked ruff format --check tools
	uv run --locked ty check --error-on-warning

fmtpython:
	uv run --locked ruff check --fix tools
	uv run --locked ruff format tools

generate:
	bash tools/schema/generate.sh

docs:
	$(PYTHON) tools/docs/resources.py

fmt: fmtpython
	gofmt -s -w -e .

test: lintpython testdiscovery testschema testmaintenance testcontracts testchr testdocs
	GOTOOLCHAIN=go1.25.8 TF_ACC_TERRAFORM_VERSION=1.14.0 go test -v -race -cover -timeout=300s ./...

testdiscovery:
	$(PYTHON) -m unittest discover -s tools/schema/discover -v

testschema:
	$(PYTHON) -m unittest discover -s tools/schema/normalize -v
	$(PYTHON) -m unittest discover -s tools/schema/validate -v

testcontracts:
	$(PYTHON) -m unittest discover -s tools/contracts -v

testmaintenance:
	$(PYTHON) -m unittest discover -s tools/maintenance -v

maintenance:
	$(PYTHON) tools/maintenance/maintain.py

testchr:
	$(PYTHON) -m unittest discover -s tools/chr -v

testdocs:
	$(PYTHON) tools/docs/resources.py --check
	$(PYTHON) -m unittest discover -s tools/docs -v

testacc:
	$(PYTHON) tools/chr/chr.py test

.PHONY: fmt fmtpython lint lintpython test testdiscovery testschema testmaintenance testcontracts testchr testdocs maintenance testacc build install generate docs
