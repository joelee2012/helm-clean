#!/usr/bin/env sh
# this was copied from git@github.com:databus23/helm-diff
if [ "$HELM_DEBUG" = true ]; then
  set -x
  env | sort
fi

REPO_URL=$(git remote get-url origin)
PROJECT_NAME=${HELM_PLUGIN_DIR##*/}
PROJECT_GH=$(echo "${REPO_URL##*:}" | sed -r 's!.git$!!g')
export GREP_COLOR="never"

# Convert HELM_BIN and HELM_PLUGIN_DIR to unix if cygpath is
# available. This is the case when using MSYS2 or Cygwin
# on Windows where helm returns a Windows path but we
# need a Unix path

if command -v cygpath >/dev/null 2>&1; then
  HELM_BIN="$(cygpath -u "${HELM_BIN}")"
  HELM_PLUGIN_DIR="$(cygpath -u "${HELM_PLUGIN_DIR}")"
fi

SCRIPT_MODE="install"
if [ "$1" = "-u" ]; then
  SCRIPT_MODE="update"
fi

# initArch discovers the architecture for this system.
initArch() {
  ARCH=$(uname -m)
  case $ARCH in
  armv5*) ARCH="armv5" ;;
  armv6*) ARCH="armv6" ;;
  armv7*) ARCH="armv7" ;;
  aarch64) ARCH="arm64" ;;
  x86) ARCH="386" ;;
  x86_64) ARCH="amd64" ;;
  i686) ARCH="386" ;;
  i386) ARCH="386" ;;
  *)
    echo echo "Arch '$ARCH' not supported!" >&2
    exit 1
    ;;
  esac
}

# initOS discovers the operating system for this system.
initOS() {
  OS=$(uname -s)
  case "$OS" in
  Windows_NT) OS='windows' ;;
  # Msys support
  MSYS*) OS='windows' ;;
  # Minimalist GNU for Windows
  MINGW*) OS='windows' ;;
  CYGWIN*) OS='windows' ;;
  Darwin) OS='macos' ;;
  Linux) OS='linux' ;;
  *)
    echo echo "Arch '$OS' not supported!" >&2
    exit 1
    ;;
  esac
}

# Temporary dir
mkTempDir() {
  HELM_TMP="$(mktemp -d -t "${PROJECT_NAME}-XXXXXX")"
}
rmTempDir() {
  if [ -d "${HELM_TMP:-/tmp/${PROJECT_NAME}-tmp}" ]; then
    rm -rf "${HELM_TMP:-/tmp/${PROJECT_NAME}-tmp}"
  fi
}

# downloadFile downloads the latest binary package and also the checksum
# for that binary.
downloadFile() {
  if command -v curl >/dev/null 2>&1; then
    DOWNLOAD_CMD="curl -sSf -L"
  elif command -v wget >/dev/null 2>&1; then
    DOWNLOAD_CMD="wget -q -O -"
  else
    echo "Either curl or wget is required"
    exit 1
  fi
  version=$(git -C "$HELM_PLUGIN_DIR" describe --tags --exact-match 2>/dev/null | sed 's/^v//g' || :)
  if [ "$SCRIPT_MODE" = "install" ] && [ -n "$version" ]; then
    DOWNLOAD_URL="https://github.com/$PROJECT_GH/releases/download/v$version/${PROJECT_NAME}_${version}_${OS}_${ARCH}.tar.gz"
  else
    version=$(curl -sL "https://api.github.com/repos/$PROJECT_GH/releases/latest" | grep tag_name | head -1 | sed -r 's/[^:]*: "v([^"]*).*/\1/g')
    DOWNLOAD_URL="https://github.com/$PROJECT_GH/releases/download/v$version/${PROJECT_NAME}_${version}_${OS}_${ARCH}.tar.gz"
  fi
  PLUGIN_TMP_FILE="${HELM_TMP}/${PROJECT_NAME}.tgz"
  echo "Downloading $DOWNLOAD_URL"
  $DOWNLOAD_CMD "$DOWNLOAD_URL" >"$PLUGIN_TMP_FILE"
}

# installFile verifies the SHA256 for the file, then unpacks and
# installs it.
installFile() {
  tar xzf "$PLUGIN_TMP_FILE" -C "$HELM_TMP"
  echo "Preparing to install into ${HELM_PLUGIN_DIR}"
  cp "$HELM_TMP/*" "$HELM_PLUGIN_DIR/"
}

# exit_trap is executed if on exit (error or not).
exit_trap() {
  result=$?
  rmTempDir
  if [ "$result" != "0" ]; then
    echo "Failed to install $PROJECT_NAME"
    printf '\tFor support, go to ${REPO_URL}.\n'
  fi
  exit $result
}

# testVersion tests the installed client to make sure it is working.
testVersion() {
  set +e
  echo "${PROJECT_NAME} installed into $HELM_PLUGIN_DIR"
  "$HELM_BIN" "$HELM_PLUGIN_NAME" -h
  set -e
}

# Execution

#Stop execution on any error
trap "exit_trap" EXIT
set -e
initArch
initOS
mkTempDir
downloadFile
installFile
testVersion
