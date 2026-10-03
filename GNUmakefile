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
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test: testdiscovery testschema
	GOTOOLCHAIN=go1.25.8 TF_ACC_TERRAFORM_VERSION=1.14.0 go test -v -race -cover -timeout=120s ./...

testdiscovery:
	python3 -m unittest discover -s tools/schema/discover -v

testschema:
	python3 -m unittest discover -s tools/schema/normalize -v
	python3 -m unittest discover -s tools/schema/validate -v

testacc:
	python3 tools/chr/chr.py test

.PHONY: fmt lint test testdiscovery testschema testacc build install generate docs
