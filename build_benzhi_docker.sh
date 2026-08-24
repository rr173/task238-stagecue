#!/usr/bin/env bash
# 用法: bash build_benzhi_docker.sh <镜像名> <平台>
# 例:  bash build_benzhi_docker.sh my-project linux/amd64
set -euo pipefail

IMAGE_NAME="${1:-my-project}"
PLATFORM="${2:-linux/amd64}"

if [[ -z "${IMAGE_NAME}" || -z "${PLATFORM}" ]]; then
  echo "usage: bash build_benzhi_docker.sh <镜像名> <平台>" >&2
  exit 2
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

echo "building ${IMAGE_NAME} for ${PLATFORM} ..."
docker buildx build --platform "${PLATFORM}" -f benzhi.Dockerfile -t "${IMAGE_NAME}:${PLATFORM##*/}" .

echo "done: ${IMAGE_NAME}:${PLATFORM##*/}"
