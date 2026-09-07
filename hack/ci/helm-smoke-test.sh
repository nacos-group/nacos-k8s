#!/usr/bin/env bash
# Helm smoke test for nacos-k8s.
# Usage: NACOS_VERSION=v3.2.3 MODE=standalone STORAGE=embedded ./hack/ci/helm-smoke-test.sh
#
# Required env vars:
#   NACOS_VERSION  - Nacos image tag (e.g. v2.5.3, v3.2.3)
#   MODE           - standalone or cluster
# Optional:
#   STORAGE        - embedded (default) or mysql
#   AUTH_MODE      - unset (default), true, or false
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

: "${NACOS_VERSION:?NACOS_VERSION is required}"
: "${MODE:?MODE is required (standalone or cluster)}"
: "${STORAGE:=embedded}"
: "${AUTH_MODE:=unset}"

case "${AUTH_MODE}" in
  unset|true|false) ;;
  *)
    echo "ERROR: AUTH_MODE must be unset, true, or false"
    exit 1
    ;;
esac

RELEASE_NAME="nacos-ci"
NAMESPACE="default"
HELM_CHART="${REPO_ROOT}/helm"
HELM_BIN="${HELM_BIN:-helm}"
AUTH_TEST_DIR="$(mktemp -d)"
CLIENT_PORT_FORWARD_PID=""
if [[ "${MODE}" == "cluster" ]]; then
  TIMEOUT="600s"
else
  TIMEOUT="300s"
fi

cleanup_client_port_forward() {
  if [[ -n "${CLIENT_PORT_FORWARD_PID}" ]]; then
    kill "${CLIENT_PORT_FORWARD_PID}" 2>/dev/null || true
    wait "${CLIENT_PORT_FORWARD_PID}" 2>/dev/null || true
    CLIENT_PORT_FORWARD_PID=""
  fi
}

cleanup_local_files() {
  cleanup_client_port_forward
  rm -rf "${AUTH_TEST_DIR}"
}

trap cleanup_local_files EXIT

# Determine major version (2 or 3) from tag like v2.5.3 or v3.2.3
MAJOR_VERSION="${NACOS_VERSION#v}"
MAJOR_VERSION="${MAJOR_VERSION%%.*}"

if [[ "${MAJOR_VERSION}" == "2" ]]; then
  VALUES_FILE="${SCRIPT_DIR}/values-nacos-2x.yaml"
  HEALTH_PORT=8848
  HEALTH_PATH="/nacos/"
elif [[ "${MAJOR_VERSION}" == "3" ]]; then
  VALUES_FILE="${SCRIPT_DIR}/values-nacos-3x.yaml"
  HEALTH_PORT=8080
  HEALTH_PATH="/v3/console/health/readiness"
else
  echo "ERROR: Unsupported Nacos major version: ${MAJOR_VERSION} (from ${NACOS_VERSION})"
  exit 1
fi

echo "=== Helm Smoke Test ==="
echo "  Nacos version : ${NACOS_VERSION} (${MAJOR_VERSION}.x)"
echo "  Mode          : ${MODE}"
echo "  Storage       : ${STORAGE}"
echo "  Auth mode     : ${AUTH_MODE}"
echo "  Values file   : ${VALUES_FILE}"
echo "  Health check  : :${HEALTH_PORT}${HEALTH_PATH}"
echo ""

# --- MySQL setup (standalone + mysql only) ---
setup_mysql() {
  echo ">>> Deploying MySQL..."
  kubectl apply -f "${SCRIPT_DIR}/mysql.yaml"
  echo ">>> Waiting for MySQL to be ready..."
  kubectl wait --for=condition=Ready pod -l app=mysql --timeout="${TIMEOUT}"

  echo ">>> Downloading Nacos schema SQL..."
  local clean_version="${NACOS_VERSION#v}"
  clean_version="${clean_version%-*}"
  local new_schema_url="https://raw.githubusercontent.com/alibaba/nacos/${clean_version}/plugin-default-impl/nacos-default-datasource-plugin/nacos-datasource-plugin-mysql/src/main/resources/META-INF/mysql-schema.sql"
  local old_schema_url="https://raw.githubusercontent.com/alibaba/nacos/${clean_version}/distribution/conf/mysql-schema.sql"
  local sql_file="/tmp/nacos-schema.sql"

  if ! curl -sSL --fail "${new_schema_url}" -o "${sql_file}" 2>/dev/null; then
    echo ">>> New path not found, trying legacy path..."
    curl -sSL --fail "${old_schema_url}" -o "${sql_file}"
  fi
  if [[ ! -s "${sql_file}" ]]; then
    echo "ERROR: Failed to download schema from both paths"
    exit 1
  fi

  echo ">>> Importing schema into MySQL..."
  local mysql_pod
  mysql_pod=$(kubectl get pods -l app=mysql -o jsonpath='{.items[0].metadata.name}')

  # readinessProbe (mysqladmin ping) passes before the server fully accepts SQL connections
  echo ">>> Waiting for MySQL to accept connections..."
  local retries=0
  until kubectl exec "${mysql_pod}" -- mysql -h 127.0.0.1 -uroot -proot -e "SELECT 1" &>/dev/null; do
    retries=$((retries + 1))
    if [[ ${retries} -ge 30 ]]; then
      echo "ERROR: MySQL not accepting connections after 30 retries"
      return 1
    fi
    sleep 2
  done

  kubectl cp "${sql_file}" "${mysql_pod}:/tmp/nacos-schema.sql"
  kubectl exec "${mysql_pod}" -- mysql -h 127.0.0.1 -uroot -proot nacos -e "source /tmp/nacos-schema.sql"
  echo ">>> MySQL setup complete."
}

cleanup_mysql() {
  echo ">>> Cleaning up MySQL..."
  kubectl delete -f "${SCRIPT_DIR}/mysql.yaml" --ignore-not-found --wait=false
}

# --- Helm install ---
helm_install() {
  local extra_args=()

  extra_args+=(--set "nacos.image.tag=${NACOS_VERSION}")
  extra_args+=(--set "global.mode=${MODE}")

  if [[ "${AUTH_MODE}" != "unset" ]]; then
    extra_args+=(--set "nacos.auth.enabled=${AUTH_MODE}")
  fi

  if [[ "${MODE}" == "cluster" ]]; then
    extra_args+=(--set "nacos.replicaCount=3")
    extra_args+=(--set "nacos.probe.startupDelaySeconds=180")
  fi

  if [[ "${STORAGE}" == "mysql" ]]; then
    extra_args+=(--set "nacos.storage.type=mysql")
    extra_args+=(--set "nacos.storage.db.host=mysql")
    extra_args+=(--set "nacos.storage.db.name=nacos")
    extra_args+=(--set "nacos.storage.db.port=3306")
    extra_args+=(--set "nacos.storage.db.username=nacos")
    extra_args+=(--set "nacos.storage.db.password=nacos")
    extra_args+=(--set "nacos.storage.db.param=characterEncoding=utf8\&connectTimeout=1000\&socketTimeout=3000\&autoReconnect=true\&useSSL=false\&allowPublicKeyRetrieval=true")
  fi

  echo ">>> Installing Helm chart..."
  echo "    helm install ${RELEASE_NAME} ${HELM_CHART} -f ${VALUES_FILE} ${extra_args[*]}"
  "${HELM_BIN}" install "${RELEASE_NAME}" "${HELM_CHART}" \
    -f "${VALUES_FILE}" \
    "${extra_args[@]}"
}

# --- Authentication deployment contract ---
auth_secret_name() {
  local statefulset=$1
  kubectl get statefulset "${statefulset}" \
    -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="NACOS_AUTH_TOKEN")].valueFrom.secretKeyRef.name}'
}

auth_secret_payload() {
  local secret_name=$1
  local key
  for key in token identity-key identity-value; do
    kubectl get secret "${secret_name}" -o "jsonpath={.data['${key}']}"
    printf '\n'
  done
}

verify_auth_deployment() {
  echo ">>> Verifying authentication deployment contract..."

  local statefulset="${RELEASE_NAME}"

  local rendered_auth_value
  rendered_auth_value=$(kubectl get statefulset "${statefulset}" \
    -o jsonpath='{.spec.template.spec.containers[0].env[?(@.name=="NACOS_AUTH_ENABLE")].value}')
  if [[ "${AUTH_MODE}" == "unset" && -n "${rendered_auth_value}" ]]; then
    echo "ERROR: unset auth mode rendered NACOS_AUTH_ENABLE=${rendered_auth_value}"
    return 1
  fi
  if [[ "${AUTH_MODE}" != "unset" && "${rendered_auth_value}" != "${AUTH_MODE}" ]]; then
    echo "ERROR: expected NACOS_AUTH_ENABLE=${AUTH_MODE}, got ${rendered_auth_value:-<unset>}"
    return 1
  fi

  local secret_name
  secret_name=$(auth_secret_name "${statefulset}")
  if [[ -z "${secret_name}" ]]; then
    echo "ERROR: NACOS_AUTH_TOKEN does not reference a Secret"
    return 1
  fi
  local key
  for key in token identity-key identity-value; do
    if [[ -z "$(kubectl get secret "${secret_name}" -o "jsonpath={.data['${key}']}")" ]]; then
      echo "ERROR: ${secret_name} does not contain non-empty ${key} data"
      return 1
    fi
  done

  echo ">>> Authentication environment and Secret references are valid."
}

verify_auth_secret_stability() {
  echo ">>> Verifying generated credentials remain stable across a no-op upgrade..."

  local statefulset="${RELEASE_NAME}"
  local secret_name
  secret_name=$(auth_secret_name "${statefulset}")
  local secret_before
  local checksum_before
  local revision_before
  secret_before=$(auth_secret_payload "${secret_name}")
  checksum_before=$(kubectl get statefulset "${statefulset}" \
    -o go-template='{{ index .spec.template.metadata.annotations "checksum/auth-config" }}')
  revision_before=$(kubectl get statefulset "${statefulset}" -o jsonpath='{.status.currentRevision}')

  "${HELM_BIN}" upgrade "${RELEASE_NAME}" "${HELM_CHART}" --reuse-values \
    --wait --timeout="${TIMEOUT}"

  local secret_after
  local checksum_after
  local revision_after
  secret_after=$(auth_secret_payload "${secret_name}")
  checksum_after=$(kubectl get statefulset "${statefulset}" \
    -o go-template='{{ index .spec.template.metadata.annotations "checksum/auth-config" }}')
  revision_after=$(kubectl get statefulset "${statefulset}" -o jsonpath='{.status.currentRevision}')

  if [[ "${secret_before}" != "${secret_after}" ]]; then
    echo "ERROR: generated authentication credentials changed during a no-op upgrade"
    return 1
  fi
  if [[ "${checksum_before}" != "${checksum_after}" ]]; then
    echo "ERROR: auth checksum changed during a no-op upgrade"
    return 1
  fi
  if [[ "${revision_before}" != "${revision_after}" ]]; then
    echo "ERROR: StatefulSet rolled during a no-op auth upgrade"
    return 1
  fi

  echo ">>> Authentication credentials and StatefulSet revision remained stable."
}

# --- Wait for pods ---
wait_for_pods() {
  echo ">>> Waiting for Nacos pods to be ready (timeout: ${TIMEOUT})..."
  local expected_pods=1
  if [[ "${MODE}" == "cluster" ]]; then
    expected_pods=3
  fi

  # kubectl wait fails immediately if no pods exist yet; poll until at least one appears
  echo ">>> Waiting for pods to be created..."
  local elapsed=0
  while [[ $(kubectl get pods -l app.kubernetes.io/name=nacos -o name 2>/dev/null | wc -l | tr -d ' ') -lt 1 ]]; do
    if [[ ${elapsed} -ge 120 ]]; then
      echo "ERROR: No pods created after 120s"
      return 1
    fi
    sleep 5
    elapsed=$((elapsed + 5))
  done

  kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=nacos \
    --timeout="${TIMEOUT}"

  local ready_count
  ready_count=$(kubectl get pods -l app.kubernetes.io/name=nacos \
    --field-selector=status.phase=Running -o name | wc -l | tr -d ' ')

  if [[ "${ready_count}" -lt "${expected_pods}" ]]; then
    echo "ERROR: Expected ${expected_pods} ready pods, got ${ready_count}"
    return 1
  fi
  echo ">>> ${ready_count}/${expected_pods} Nacos pods ready."
}

# --- Smoke verification ---
smoke_verify() {
  echo ">>> Running smoke verification..."

  local pod
  pod=$(kubectl get pods -l app.kubernetes.io/name=nacos -o jsonpath='{.items[0].metadata.name}')

  kubectl port-forward "${pod}" "${HEALTH_PORT}:${HEALTH_PORT}" &
  local pf_pid=$!
  sleep 3

  local http_code
  http_code=$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:${HEALTH_PORT}${HEALTH_PATH}" || true)

  kill "${pf_pid}" 2>/dev/null || true
  wait "${pf_pid}" 2>/dev/null || true

  if [[ "${http_code}" == "200" ]]; then
    echo ">>> Health check passed: HTTP ${http_code}"
  else
    echo "ERROR: Health check failed: HTTP ${http_code} (expected 200)"
    echo "  URL: http://localhost:${HEALTH_PORT}${HEALTH_PATH}"
    return 1
  fi
}

expected_client_auth_state() {
  if [[ "${AUTH_MODE}" == "true" ]]; then
    echo enabled
  else
    echo disabled
  fi
}

request_code() {
  local output_file=$1
  shift
  curl -sS -o "${output_file}" -w '%{http_code}' "$@" || true
}

verify_v3_auth_behavior() {
  if [[ "${MAJOR_VERSION}" != "3" || "${AUTH_MODE}" == "unset" ]]; then
    return
  fi

  echo ">>> Verifying Client, Admin, and Console authentication behavior..."
  local expected_state
  expected_state=$(expected_client_auth_state)
  local pod
  pod=$(kubectl get pods -l app.kubernetes.io/name=nacos -o jsonpath='{.items[0].metadata.name}')

  kubectl port-forward "${pod}" 18848:8848 18080:8080 \
    > "${AUTH_TEST_DIR}/port-forward.log" 2>&1 &
  CLIENT_PORT_FORWARD_PID=$!
  local retries=0
  until curl -sS -o /dev/null "http://127.0.0.1:18848/nacos/"; do
    retries=$((retries + 1))
    if [[ "${retries}" -ge 30 ]]; then
      echo "ERROR: Client port-forward did not become ready"
      return 1
    fi
    sleep 1
  done

  local login_code
  local attempt
  local admin_setup_code
  admin_setup_code=$(request_code "${AUTH_TEST_DIR}/admin-setup.json" -X POST \
    "http://127.0.0.1:18848/nacos/v3/auth/user/admin" \
    --data-urlencode 'password=nacos')
  if [[ "${admin_setup_code}" != "200" ]]; then
    echo "ERROR: administrator initialization returned HTTP ${admin_setup_code}"
    return 1
  fi
  local admin_result_code
  admin_result_code=$(python3 -c \
    'import json,sys; print(json.load(sys.stdin).get("code", ""))' \
    < "${AUTH_TEST_DIR}/admin-setup.json")
  if [[ "${admin_result_code}" != "0" && "${admin_result_code}" != "409" ]]; then
    echo "ERROR: administrator initialization returned code ${admin_result_code:-<missing>}"
    return 1
  fi

  login_code=""
  for attempt in {1..30}; do
    login_code=$(request_code "${AUTH_TEST_DIR}/login.json" -X POST \
      "http://127.0.0.1:18848/nacos/v3/auth/user/login" \
      --data-urlencode 'username=nacos' --data-urlencode 'password=nacos')
    if [[ "${login_code}" == "200" ]]; then
      break
    fi
    sleep 1
  done
  if [[ "${login_code}" != "200" ]]; then
    echo "ERROR: administrator login returned HTTP ${login_code}"
    return 1
  fi

  local access_token
  access_token=$(python3 -c \
    'import json,sys; value=json.load(sys.stdin); print(value.get("accessToken", ""))' \
    < "${AUTH_TEST_DIR}/login.json")
  if [[ -z "${access_token}" ]]; then
    echo "ERROR: administrator login did not return accessToken"
    return 1
  fi

  local data_id="helm-auth-${AUTH_MODE}-${MODE}-${STORAGE}"
  local content="authenticated-${AUTH_MODE}"
  local publish_code
  publish_code=$(request_code "${AUTH_TEST_DIR}/publish.json" -X POST \
    "http://127.0.0.1:18848/nacos/v3/admin/cs/config" \
    -H "accessToken: ${access_token}" \
    --data-urlencode "dataId=${data_id}" \
    --data-urlencode 'groupName=DEFAULT_GROUP' \
    --data-urlencode "content=${content}")
  if [[ "${publish_code}" != "200" ]]; then
    echo "ERROR: authenticated config publish returned HTTP ${publish_code}"
    return 1
  fi

  local authorized_code
  authorized_code=""
  for attempt in {1..30}; do
    authorized_code=$(request_code "${AUTH_TEST_DIR}/authorized-client.json" -G \
      "http://127.0.0.1:18848/nacos/v3/client/cs/config" \
      -H "accessToken: ${access_token}" \
      --data-urlencode "dataId=${data_id}" \
      --data-urlencode 'groupName=DEFAULT_GROUP')
    if [[ "${authorized_code}" == "200" ]] \
      && grep -Fq "${content}" "${AUTH_TEST_DIR}/authorized-client.json"; then
      break
    fi
    sleep 1
  done
  if [[ "${authorized_code}" != "200" ]] \
    || ! grep -Fq "${content}" "${AUTH_TEST_DIR}/authorized-client.json"; then
    echo "ERROR: authenticated Client API request did not return the published content"
    return 1
  fi

  local anonymous_client_code
  anonymous_client_code=$(request_code "${AUTH_TEST_DIR}/anonymous-client.json" -G \
    "http://127.0.0.1:18848/nacos/v3/client/cs/config" \
    --data-urlencode "dataId=${data_id}" \
    --data-urlencode 'groupName=DEFAULT_GROUP')
  if [[ "${expected_state}" == "enabled" && "${anonymous_client_code}" != "403" ]]; then
    echo "ERROR: anonymous Client API returned HTTP ${anonymous_client_code}; expected 403"
    return 1
  fi
  if [[ "${expected_state}" == "disabled" && "${anonymous_client_code}" != "200" ]]; then
    echo "ERROR: anonymous Client API returned HTTP ${anonymous_client_code}; expected 200"
    return 1
  fi
  if [[ "${expected_state}" == "disabled" ]] \
    && ! grep -Fq "${content}" "${AUTH_TEST_DIR}/anonymous-client.json"; then
    echo "ERROR: anonymous Client API did not return the published content"
    return 1
  fi

  local anonymous_admin_code
  anonymous_admin_code=$(request_code "${AUTH_TEST_DIR}/anonymous-admin.json" -G \
    "http://127.0.0.1:18848/nacos/v3/admin/cs/config/list" \
    --data-urlencode 'namespaceId=public' \
    --data-urlencode 'pageNo=1' \
    --data-urlencode 'pageSize=10')
  if [[ "${anonymous_admin_code}" != "403" ]]; then
    echo "ERROR: anonymous Admin API returned HTTP ${anonymous_admin_code}; expected 403"
    return 1
  fi

  local anonymous_console_code
  anonymous_console_code=$(request_code "${AUTH_TEST_DIR}/anonymous-console.json" -G \
    "http://127.0.0.1:18080/v3/console/core/namespace" \
    --data-urlencode 'namespaceId=public')
  if [[ "${anonymous_console_code}" != "403" ]]; then
    echo "ERROR: anonymous Console API returned HTTP ${anonymous_console_code}; expected 403"
    return 1
  fi

  cleanup_client_port_forward
  echo ">>> Authentication behavior is valid (${expected_state})."
}

# --- Diagnostics (called on failure) ---
collect_diagnostics() {
  local diag_dir="${DIAGNOSTICS_DIR:-/tmp/diagnostics}"
  mkdir -p "${diag_dir}"

  echo ">>> Collecting diagnostics to ${diag_dir}..."
  kubectl get pods -o wide > "${diag_dir}/pods.txt" 2>&1 || true
  kubectl describe pods -l app.kubernetes.io/name=nacos > "${diag_dir}/describe.txt" 2>&1 || true

  for pod in $(kubectl get pods -l app.kubernetes.io/name=nacos -o name 2>/dev/null); do
    local pod_name="${pod#pod/}"
    kubectl logs "${pod}" --all-containers > "${diag_dir}/logs-${pod_name}.txt" 2>&1 || true
    kubectl logs "${pod}" --all-containers --previous > "${diag_dir}/logs-${pod_name}-previous.txt" 2>&1 || true
  done

  echo ">>> Diagnostics saved to ${diag_dir}"
}

# --- Main ---
trap 'collect_diagnostics' ERR

if [[ "${STORAGE}" == "mysql" ]]; then
  setup_mysql
fi

helm_install
wait_for_pods
verify_auth_deployment
verify_auth_secret_stability
smoke_verify
verify_v3_auth_behavior

echo ""
echo "=== PASSED: ${MODE} / ${STORAGE} / ${NACOS_VERSION} ==="

# Cleanup for next round (when running multiple storage types)
"${HELM_BIN}" uninstall "${RELEASE_NAME}" --wait 2>/dev/null || true
if [[ "${STORAGE}" == "mysql" ]]; then
  cleanup_mysql
fi
