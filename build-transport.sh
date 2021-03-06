#!/bin/bash
# Copyright 2019-2020 VMware, Inc.
# SPDX-License-Identifier: BSD-2-Clause
#
# This script will build the binary sample application for transport
#
COLOR_RESET="\033[0m"
COLOR_RED="\033[38;5;9m"
COLOR_LIGHTCYAN="\033[1;36m"
COLOR_LIGHTGREEN="\033[1;32m"

COMMANDS=(transport-go)
OUT_DIR=${OUT_DIR:-./}
BUILD_TIME=`date | sed -e 's/ /_/g'`
TARGET_OS=${TARGET_OS:-darwin}
TARGET_ARCH=${TARGET_ARCH:-amd64}

GIT_HASH=${GIT_HASH:-$(git rev-parse --short HEAD)}
VERSION=v${MAJOR_VER}.${MINOR_VER}

error() {
    echo -e "${COLOR_RED}ERROR: $1${COLOR_RESET}" >&2
    exit 1
}

warn() {
    echo -e "${COLOR_RED}WARNING: $1${COLOR_RESET}"
}

info() {
    echo -e "${COLOR_LIGHTCYAN}$1${COLOR_RESET}"
}

success() {
    echo -e "${COLOR_LIGHTGREEN}$1${COLOR_RESET}"
}

_trap() {
  echo interrupted >&2
  exit 1
}

build() {
    local CMD=$1
    info "Building ${CMD} for ${TARGET_OS} ${TARGET_ARCH}..."

    if [[ "$TARGET_OS" = "darwin" || "$TARGET_OS" = "linux" ]] ; then
        local OUTPUT_FILE="$CMD"
    else
        local OUTPUT_FILE="${CMD}.exe"
    fi
    local OUTPUT_PATH="${OUT_DIR}/${OUTPUT_FILE}"

    # build
    go build -ldflags "-X main.BuildTime=${BUILD_TIME} -X main.Version=${VERSION}-${GIT_HASH}" \
             -o $OUTPUT_PATH transport.go sample_services.go sample_vm_service.go
    if [ $? -ne 0 ] ; then
        error "Build Failed!"
    fi

    chmod +x ${OUTPUT_PATH}

}

trap '_trap' SIGINT SIGTERM

while getopts ":o:a:" flag ; do
    case $flag in
        o)
            TARGET_OS=${OPTARG}
            ;;
        a)
            TARGET_ARCH=${OPTARG}
            ;;
        *)
            echo "Usage: $0 [-a architecture - 386, amd64] [-o target OS - windows, darwin, linux]" >&2
            exit 1
            ;;
    esac
done

# ensure output dir exist
mkdir -p ${OUT_DIR}

for CMD in ${COMMANDS[@]} ; do
    GOOS=${TARGET_OS} GOARCH=${TARGET_ARCH} build $CMD
done

