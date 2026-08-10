#!/usr/bin/env bash
# Helm smoke test for nacos-k8s.
# Usage: NACOS_VERSION=v3.2.3 MODE=standalone STORAGE=embedded ./hack/ci/helm-smoke-test.sh
#
# Required env vars:
#   NACOS_VERSION  - Nacos image tag (e.g. v2.5.3, v3.2.3)
#   MODE           - standalone or cluster
# Optional:
#   STORAGE        - embedded (default) or mysql (standalone only)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

: "${NACOS_VERSION:?NACOS_VERSION is required}"
: "${MODE:?MODE is required (standalone or cluster)}"
: "${STORAGE:=embedded}"

RELEASE_NAME="nacos-ci"
NAMESPACE="default"
HELM_CHART="${REPO_ROOT}/helm"
TIMEOUT="300s"

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

  if [[ "${MODE}" == "cluster" ]]; then
    extra_args+=(--set "nacos.replicaCount=3")
  fi

  if [[ "${STORAGE}" == "mysql" ]]; then
    extra_args+=(--set "nacos.storage.type=mysql")
    extra_args+=(--set "nacos.storage.db.host=mysql")
    extra_args+=(--set "nacos.storage.db.name=nacos")
    extra_args+=(--set "nacos.storage.db.port=3306")
    extra_args+=(--set "nacos.storage.db.username=nacos")
    extra_args+=(--set "nacos.storage.db.password=nacos")
    extra_args+=(--set "nacos.storage.db.param=characterEncoding=utf8\&connectTimeout=1000\&socketTimeout=3000\&autoReconnect=true\&useSSL=false")
  fi

  echo ">>> Installing Helm chart..."
  echo "    helm install ${RELEASE_NAME} ${HELM_CHART} -f ${VALUES_FILE} ${extra_args[*]}"
  helm install "${RELEASE_NAME}" "${HELM_CHART}" \
    -f "${VALUES_FILE}" \
    "${extra_args[@]}"
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
smoke_verify

echo ""
echo "=== PASSED: ${MODE} / ${STORAGE} / ${NACOS_VERSION} ==="

# Cleanup for next round (when running multiple storage types)
helm uninstall "${RELEASE_NAME}" --wait 2>/dev/null || true
if [[ "${STORAGE}" == "mysql" ]]; then
  cleanup_mysql
fi
