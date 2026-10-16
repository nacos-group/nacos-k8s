#!/usr/bin/env bash
# Helm render tests for chart-managed and user-supplied environment variables.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
HELM_CHART="${REPO_ROOT}/helm"
HELM_BIN="${HELM_BIN:-helm}"
TEST_DIR="$(mktemp -d)"
trap 'rm -rf "${TEST_DIR}"' EXIT

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

render() {
  local name=$1
  shift
  "${HELM_BIN}" template "${name}" "${HELM_CHART}" "$@" > "${TEST_DIR}/${name}.yaml"
}

# Prints the value of the given env var, once per occurrence.
env_values() {
  local env_name=$1
  local file=$2
  awk -v name="${env_name}" '
    $0 ~ "^[[:space:]]*- name: " name "$" {
      getline
      sub(/^[[:space:]]*value: /, "")
      gsub(/"/, "")
      print
    }
  ' "${file}"
}

assert_env() {
  local expected=$1
  local env_name=$2
  local name=$3
  local actual
  actual=$(env_values "${env_name}" "${TEST_DIR}/${name}.yaml" | paste -sd, -)
  if [[ "${actual}" != "${expected}" ]]; then
    fail "${name}: expected ${env_name}='${expected}', got '${actual}'"
  fi
}

render v3-cluster-embedded --set global.mode=cluster
render v3-cluster-embedded-disabled --set global.mode=cluster --set nacos.storage.embedded.migrateRetryTimes=0
render v3-standalone-embedded
render v3-cluster-mysql --set global.mode=cluster --set nacos.storage.type=mysql
render v2-cluster-embedded --set global.mode=cluster --set nacos.image.tag=v2.5.4
render extra-env --set global.mode=cluster \
  --set 'nacos.extraEnv[0].name=NACOS_CONFIG_NAMESPACE_COMPATIBLE_MODE' \
  --set-string 'nacos.extraEnv[0].value=false'

# The migration retry override is only rendered for Nacos 3.x cluster + embedded.
assert_env "60" NACOS_NAMESPACE_MIGRATE_RETRY_TIMES v3-cluster-embedded
assert_env "" NACOS_NAMESPACE_MIGRATE_RETRY_TIMES v3-cluster-embedded-disabled
assert_env "" NACOS_NAMESPACE_MIGRATE_RETRY_TIMES v3-standalone-embedded
assert_env "" NACOS_NAMESPACE_MIGRATE_RETRY_TIMES v3-cluster-mysql
assert_env "" NACOS_NAMESPACE_MIGRATE_RETRY_TIMES v2-cluster-embedded

# extraEnv is appended to the container environment.
assert_env "false" NACOS_CONFIG_NAMESPACE_COMPATIBLE_MODE extra-env
assert_env "" NACOS_CONFIG_NAMESPACE_COMPATIBLE_MODE v3-cluster-embedded

if "${HELM_BIN}" template invalid "${HELM_CHART}" \
  --set nacos.storage.embedded.migrateRetryTimes=-1 > /dev/null 2>&1; then
  fail "schema accepted a negative nacos.storage.embedded.migrateRetryTimes"
fi

echo "=== PASSED: Helm environment render tests ==="
