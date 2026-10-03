#!/usr/bin/env bash
set -euo pipefail
export GOTOOLCHAIN=go1.25.8
cd "$(dirname "$0")/../.."
mkdir -p tools/bin .local internal/generated
GOBIN="$PWD/tools/bin" go install github.com/hashicorp/terraform-plugin-codegen-openapi/cmd/tfplugingen-openapi@v0.3.0
GOBIN="$PWD/tools/bin" go install github.com/hashicorp/terraform-plugin-codegen-framework/cmd/tfplugingen-framework@v0.4.1
# Arguments belong to the consumer adapter (e.g. --input/--manifest/--key).
# The maintained provider layout uses schemas/ as its output directory.
go run tools/schema/normalize/main.go "$@" --collections --output schemas
python3 tools/schema/normalize/verify.py --output schemas
stage=$(mktemp -d "$PWD/.local/codegen.XXXXXX")
trap 'rm -rf "$stage"' EXIT
# CLI exit success alone is not a gate: reject omissions, duplicates, unions,
# enum leakage and every unreviewed change to the approved resource contract.
tools/bin/tfplugingen-openapi generate --config schemas/generator-config.yml --output "$stage/provider-code-spec.json" schemas/ip-address.openapi.json
go run tools/schema/validate/main.go "$stage/provider-code-spec.json"
mkdir -p "$stage/generated"
tools/bin/tfplugingen-framework generate resources --input "$stage/provider-code-spec.json" --output "$stage/generated" --package generated
test -f "$stage/generated/ip_address_resource_gen.go"
test "$(find "$stage/generated" -type f -name '*.go' | wc -l | tr -d ' ')" = 16
for name in interface_bridge interface_bridge_port interface_vlan interface_list interface_list_member ip_pool; do
 test -f "$stage/generated/${name}_resource_gen.go"
done
gofmt -w "$stage/generated"
# Do not overwrite the last good specification/model with permissive CLI output.
mv "$stage/provider-code-spec.json" schemas/provider-code-spec.json
for file in "$stage/generated/"*.go; do mv "$file" internal/generated/; done
