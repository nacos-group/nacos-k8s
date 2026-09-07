#!/usr/bin/env bash
# Helm render tests for the Nacos authentication contract.
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

count_matching_lines() {
  local pattern=$1
  local file=$2
  grep -Ec "${pattern}" "${file}" || true
}

assert_count() {
  local expected=$1
  local pattern=$2
  local file=$3
  local actual
  actual=$(count_matching_lines "${pattern}" "${file}")
  if [[ "${actual}" != "${expected}" ]]; then
    fail "expected ${expected} line(s) matching '${pattern}' in ${file}, got ${actual}"
  fi
}

assert_auth_value() {
  local expected=$1
  local file=$2
  if ! awk -v expected="${expected}" '
    /^[[:space:]]*- name: NACOS_AUTH_ENABLE$/ {
      getline
      if ($0 ~ "value: \\\"" expected "\\\"") {
        matches++
      }
    }
    END { exit matches == 1 ? 0 : 1 }
  ' "${file}"; then
    fail "NACOS_AUTH_ENABLE=${expected} was not rendered exactly once in ${file}"
  fi
}

extract_secret_data() {
  local key=$1
  local file=$2
  awk -v key="${key}:" '
    /^data:$/ { in_data=1; next }
    in_data && $1 == key {
      gsub(/"/, "", $2)
      print $2
      exit
    }
  ' "${file}"
}

"${HELM_BIN}" lint "${HELM_CHART}"

render auth-unset
render auth-true --set nacos.auth.enabled=true --set global.mode=cluster
render auth-false --set nacos.auth.enabled=false --set nacos.storage.type=mysql
render auth-existing-secret \
  --set nacos.auth.existingSecret=user-managed-auth \
  --set nacos.auth.tokenSecretKey=custom-token \
  --set nacos.auth.identityKeySecretKey=custom-identity-key \
  --set nacos.auth.identityValueSecretKey=custom-identity-value
legacy_token='VGhpc0lzQ3VzdG9tU2VjcmV0S2V5MEluSXRzVmVyeVNhZmU='
render auth-legacy-inline \
  --set-string "nacos.authToken=${legacy_token}" \
  --set-string nacos.identityKey=legacy-key \
  --set-string nacos.identityValue=legacy-value

assert_count 0 '^[[:space:]]*- name: NACOS_AUTH_ENABLE$' "${TEST_DIR}/auth-unset.yaml"
assert_count 1 '^[[:space:]]*- name: NACOS_AUTH_ENABLE$' "${TEST_DIR}/auth-true.yaml"
assert_count 1 '^[[:space:]]*- name: NACOS_AUTH_ENABLE$' "${TEST_DIR}/auth-false.yaml"
assert_auth_value true "${TEST_DIR}/auth-true.yaml"
assert_auth_value false "${TEST_DIR}/auth-false.yaml"

for file in auth-unset auth-true auth-false auth-existing-secret auth-legacy-inline; do
  assert_count 1 '^[[:space:]]*- name: NACOS_AUTH_TOKEN$' "${TEST_DIR}/${file}.yaml"
  assert_count 1 '^[[:space:]]*- name: NACOS_AUTH_IDENTITY_KEY$' "${TEST_DIR}/${file}.yaml"
  assert_count 1 '^[[:space:]]*- name: NACOS_AUTH_IDENTITY_VALUE$' "${TEST_DIR}/${file}.yaml"
done

assert_count 1 '^kind: Secret$' "${TEST_DIR}/auth-unset.yaml"
assert_count 0 '^kind: Secret$' "${TEST_DIR}/auth-existing-secret.yaml"
assert_count 3 '^[[:space:]]*name: user-managed-auth$' "${TEST_DIR}/auth-existing-secret.yaml"
assert_count 1 '^[[:space:]]*key: "custom-token"$' "${TEST_DIR}/auth-existing-secret.yaml"
assert_count 1 '^[[:space:]]*key: "custom-identity-key"$' "${TEST_DIR}/auth-existing-secret.yaml"
assert_count 1 '^[[:space:]]*key: "custom-identity-value"$' "${TEST_DIR}/auth-existing-secret.yaml"

legacy_token_data=$(extract_secret_data token "${TEST_DIR}/auth-legacy-inline.yaml")
legacy_identity_key_data=$(extract_secret_data identity-key "${TEST_DIR}/auth-legacy-inline.yaml")
legacy_identity_value_data=$(extract_secret_data identity-value "${TEST_DIR}/auth-legacy-inline.yaml")
[[ "$(printf '%s' "${legacy_token_data}" | openssl base64 -d -A)" == "${legacy_token}" ]] \
  || fail "legacy inline authToken changed when moved into Secret data"
[[ "$(printf '%s' "${legacy_identity_key_data}" | openssl base64 -d -A)" == "legacy-key" ]] \
  || fail "legacy inline identityKey changed when moved into Secret data"
[[ "$(printf '%s' "${legacy_identity_value_data}" | openssl base64 -d -A)" == "legacy-value" ]] \
  || fail "legacy inline identityValue changed when moved into Secret data"

token_data=$(extract_secret_data token "${TEST_DIR}/auth-unset.yaml")
[[ -n "${token_data}" ]] || fail "generated token Secret data is empty"
token_value=$(printf '%s' "${token_data}" | openssl base64 -d -A)
decoded_token_length=$(printf '%s' "${token_value}" | openssl base64 -d -A | wc -c | tr -d ' ')
if [[ "${decoded_token_length}" -lt 32 ]]; then
  fail "generated Nacos token decodes to fewer than 32 bytes"
fi

if "${HELM_BIN}" template invalid "${HELM_CHART}" \
  --set-string nacos.auth.enabled=invalid >/dev/null 2>&1; then
  fail "nacos.auth.enabled accepted a non-boolean, non-null value"
fi

echo "Helm authentication render tests passed."
