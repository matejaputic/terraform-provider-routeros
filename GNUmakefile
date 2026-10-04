default: test build

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	bash tools/schema/generate.sh

docs:
	python3 tools/docs/resources.py

fmt:
	gofmt -s -w -e .

test: testdiscovery testschema testmaintenance testcontracts testchr testdocs
	GOTOOLCHAIN=go1.25.8 TF_ACC_TERRAFORM_VERSION=1.14.0 go test -v -race -cover -timeout=300s ./...

testdiscovery:
	python3 -m unittest discover -s tools/schema/discover -v

testschema:
	python3 -m unittest discover -s tools/schema/normalize -v
	python3 -m unittest discover -s tools/schema/validate -v

testcontracts:
	python3 -m unittest discover -s tools/contracts -v

testmaintenance:
	python3 -m unittest discover -s tools/maintenance -v

maintenance:
	python3 tools/maintenance/maintain.py

testchr:
	python3 -m unittest discover -s tools/chr -v

testdocs:
	python3 tools/docs/resources.py --check
	python3 -m unittest discover -s tools/docs -v

testacc:
	python3 tools/chr/chr.py test

.PHONY: fmt lint test testdiscovery testschema testmaintenance testcontracts testchr testdocs maintenance testacc build install generate docs
