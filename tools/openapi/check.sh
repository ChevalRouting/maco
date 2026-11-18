set -eu
check_dir=$(mktemp -d /private/tmp/maco-api-check.XXXXXX)
trap 'rm -rf "$check_dir"' EXIT
swag init --v3.1 -g doc.go -d ./pkg/api --parseDependency --parseInternal --requiredByDefault --outputTypes json -o "$check_dir"
python3 tools/openapi/normalize.py "$check_dir/swagger.json"
(cd web && node_modules/.bin/openapi-ts -i "$check_dir/swagger.json" -o "$check_dir/generated")
cmp pkg/api/docs/swagger.json "$check_dir/swagger.json"
cmp web/src/api/generated/sdk.gen.ts "$check_dir/generated/sdk.gen.ts"
cmp web/src/api/generated/types.gen.ts "$check_dir/generated/types.gen.ts"
go test ./pkg/api -run TestOpenAPIRoutes
