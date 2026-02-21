#!/usr/bin/env bash
set -euo pipefail

# Reproduce the full e2e-docker flow from this repository:
# build latest CLI -> prepare env -> compose up -> deploy -> pytest-only.

usage() {
  cat <<'USAGE'
Usage:
  fixtures/esb-e2e-docker/reproduce-e2e-docker.sh [options]

Options:
  --esb-repo <path>   ESB repository path (default: ../esb from esb-cli repo root)
  --clean             Run compose down -v before compose up
  --skip-deploy       Skip esb deploy
  --skip-test         Skip pytest-only run
  --skip-image-push   Skip local fixture image build/push
  -h, --help          Show this help
USAGE
}

log() {
  printf '[repro-e2e] %s\n' "$*"
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not found: $1" >&2
    exit 1
  fi
}

# Resolve repository paths relative to this script so it can run from any cwd.
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
CLI_REPO="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
DEFAULT_ESB_REPO="${CLI_REPO}/../esb"

# Default execution toggles; can be overridden by CLI options below.
ESB_REPO="${DEFAULT_ESB_REPO}"
RUN_CLEAN=0
SKIP_DEPLOY=0
SKIP_TEST=0
SKIP_IMAGE_PUSH=0

# Parse CLI options.
while [[ $# -gt 0 ]]; do
  case "$1" in
    --esb-repo)
      ESB_REPO="$2"
      shift 2
      ;;
    --clean)
      RUN_CLEAN=1
      shift
      ;;
    --skip-deploy)
      SKIP_DEPLOY=1
      shift
      ;;
    --skip-test)
      SKIP_TEST=1
      shift
      ;;
    --skip-image-push)
      SKIP_IMAGE_PUSH=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage
      exit 1
      ;;
  esac
done

# Fixed scenario settings for reproducible e2e-docker execution.
PROJECT_NAME="esb-e2e-docker"
PROFILE="e2e-docker"
COMPOSE_FILE="${ESB_REPO}/e2e/environments/e2e-docker/docker-compose.yml"
ESB_ENV_BASE="${ESB_REPO}/e2e/environments/e2e-docker/.env"
ESB_ENV_EXAMPLE="${ESB_REPO}/.env.example"
MERGED_ENV="${ESB_REPO}/.agent/e2e-docker.merged.env"
CLI_BIN="${CLI_REPO}/.agent/bin/esb-latest"
ARTIFACT_BASE="${CLI_REPO}/.agent/verify-artifacts"
TS="$(date +%Y%m%d-%H%M%S)"
ARTIFACT_ROOT="${ARTIFACT_BASE}/${TS}"
DEPLOY_LOG="${ARTIFACT_ROOT}/deploy.log"
E2E_TEST_LOG="${ESB_REPO}/.agent/e2e-test-only-${TS}.log"
IMAGE_URI="127.0.0.1:5010/esb-e2e-lambda-python:latest"
IMAGE_BUILD_CONTEXT="${SCRIPT_DIR}/image-java-python/images/lambda/python"

# Preflight: verify required repository layout and files exist.
if [[ ! -d "${ESB_REPO}" ]]; then
  echo "esb repo not found: ${ESB_REPO}" >&2
  exit 1
fi

if [[ ! -f "${COMPOSE_FILE}" ]]; then
  echo "compose file not found: ${COMPOSE_FILE}" >&2
  exit 1
fi

if [[ ! -f "${ESB_ENV_BASE}" || ! -f "${ESB_ENV_EXAMPLE}" ]]; then
  echo "required env files not found under ${ESB_REPO}" >&2
  exit 1
fi

# Preflight: verify required external commands are installed.
require_cmd go
require_cmd docker
require_cmd curl
require_cmd awk

if [[ "${SKIP_TEST}" -eq 0 ]]; then
  require_cmd uv
fi

# Ensure output directories exist for binaries, artifacts, and logs.
mkdir -p "${ESB_REPO}/.agent" "${CLI_REPO}/.agent/bin" "${ARTIFACT_ROOT}"

# Always rebuild the CLI from current workspace so the latest code is used.
log "building latest esb-cli binary: ${CLI_BIN}"
(
  cd "${CLI_REPO}"
  go build -o "${CLI_BIN}" ./cmd/esb
)
log "esb version: $("${CLI_BIN}" version)"

# Merge E2E base env + example env + explicit overrides.
# For duplicate keys, keep the last value (override wins) in original key order.
log "generating merged env: ${MERGED_ENV}"
{
  cat "${ESB_ENV_BASE}"
  cat "${ESB_ENV_EXAMPLE}"
  cat <<EOF
ESB_ENV=${PROFILE}
LOG_LEVEL=DEBUG
E2E_IMAGE_EXPECTED_LOGGER=cloudwatch.logs.python
PORT_GATEWAY_HTTPS=18443
PORT_AGENT_GRPC=15051
PORT_AGENT_METRICS=19091
PORT_S3=19000
PORT_S3_MGMT=19001
PORT_DATABASE=18000
PORT_VICTORIALOGS=19428
PORT_REGISTRY=5010
CIRCUIT_BREAKER_THRESHOLD=3
CIRCUIT_BREAKER_RECOVERY_TIMEOUT=10.0
DEFAULT_MAX_CAPACITY=3
GATEWAY_IDLE_TIMEOUT_SECONDS=15
HEARTBEAT_INTERVAL=5
ORPHAN_GRACE_PERIOD_SECONDS=30
PING_TIMEOUT=1.0
ENV=dev
AUTH_USER=test-admin
AUTH_PASS=test-secure-password
JWT_SECRET_KEY=test-secret-key-must-be-at-least-32-chars
X_API_KEY=test-api-key
RUSTFS_ACCESS_KEY=rustfsadmin
RUSTFS_SECRET_KEY=rustfsadmin
S3_PRESIGN_ENDPOINT=http://localhost:9000
PROJECT_NAME=${PROJECT_NAME}
CONFIG_DIR=${ESB_REPO}/.esb/staging/${PROJECT_NAME}/${PROFILE}/config
EOF
} | awk -F= '
/^[A-Za-z_][A-Za-z0-9_]*=/{
  key=$1
  val=substr($0, index($0, "=")+1)
  if (!(key in seen)) {
    order[++n]=key
    seen[key]=1
  }
  values[key]=val
}
END{
  for (i=1; i<=n; i++) {
    k=order[i]
    printf "%s=%s\n", k, values[k]
  }
}
' > "${MERGED_ENV}"
chmod 600 "${MERGED_ENV}"

# Optional hard reset of compose state.
if [[ "${RUN_CLEAN}" -eq 1 ]]; then
  log "clean mode: compose down -v"
  docker compose -f "${COMPOSE_FILE}" --env-file "${MERGED_ENV}" -p "${PROJECT_NAME}" down -v --remove-orphans || true
fi

# Boot runtime services used by deploy and tests.
log "starting e2e-docker compose stack"
docker compose -f "${COMPOSE_FILE}" --env-file "${MERGED_ENV}" -p "${PROJECT_NAME}" up -d --build

# Ensure internal registry is available for image-based functions.
log "starting internal registry service"
docker compose -f "${ESB_REPO}/docker-compose.infra.yml" up -d registry

log "waiting for local registry readiness"
for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:5010/v2/" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
curl -fsS "http://127.0.0.1:5010/v2/" >/dev/null

# Prepare fixture image expected by the ImageUri function.
if [[ "${SKIP_IMAGE_PUSH}" -eq 0 ]]; then
  log "building and pushing fixture image: ${IMAGE_URI}"
  docker build -t "${IMAGE_URI}" "${IMAGE_BUILD_CONTEXT}"
  docker push "${IMAGE_URI}"
else
  log "skip image build/push"
fi

# Deploy split templates and materialize a single-root artifact output.
if [[ "${SKIP_DEPLOY}" -eq 0 ]]; then
  log "running esb deploy (artifact root: ${ARTIFACT_ROOT})"
  (
    cd "${ESB_REPO}"
    "${CLI_BIN}" deploy \
      --template "${SCRIPT_DIR}/core/template.yaml" \
      --template "${SCRIPT_DIR}/image-java-python/template.yaml" \
      --template "${SCRIPT_DIR}/stateful/template.yaml" \
      --env "${PROFILE}" \
      --mode docker \
      --project "${PROJECT_NAME}" \
      --compose-file "${COMPOSE_FILE}" \
      --env-file "${MERGED_ENV}" \
      --artifact-root "${ARTIFACT_ROOT}" \
      --image-uri "lambda-image=${IMAGE_URI}" \
      --image-runtime "lambda-image=python" \
      --secret-env "${MERGED_ENV}" \
      --verbose \
      > "${DEPLOY_LOG}" 2>&1
  )
  test -f "${ARTIFACT_ROOT}/artifact.yml"
  log "deploy completed: ${DEPLOY_LOG}"
else
  log "skip deploy"
fi

# Run E2E tests in test-only mode against the already deployed environment.
if [[ "${SKIP_TEST}" -eq 0 ]]; then
  log "running pytest-only via e2e runner"
  (
    cd "${ESB_REPO}"
    uv run e2e/run_tests.py --profile "${PROFILE}" --test-only --no-live --verbose \
      | tee "${E2E_TEST_LOG}"
  )
  log "test log: ${E2E_TEST_LOG}"
else
  log "skip test"
fi

# Print pointers to key outputs for quick follow-up.
log "done"
log "artifact root: ${ARTIFACT_ROOT}"
if [[ -f "${ARTIFACT_ROOT}/artifact.yml" ]]; then
  log "artifact manifest: ${ARTIFACT_ROOT}/artifact.yml"
fi
