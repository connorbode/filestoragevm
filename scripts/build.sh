#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Load the constants
# Set the PATHS
GOPATH="$(go env GOPATH)"

# FilestorageVM root directory
FILESTORAGEVM_PATH=$( cd "$( dirname "${BASH_SOURCE[0]}" )"; cd .. && pwd )

# Set default binary directory location
binary_directory="$GOPATH/src/github.com/ava-labs/avalanchego/build/avalanchego-latest/plugins/"
name="qAyzuhzkcQQsAYQP3iibkD28DqXTS8cRsFC8PR3LuqebWVS2Q"

if [[ $# -eq 1 ]]; then
    binary_directory=$1
elif [[ $# -eq 2 ]]; then
    binary_directory=$1
    name=$2
elif [[ $# -ne 0 ]]; then
    echo "Invalid arguments to build filestoragevm. Requires either no arguments (default) or one arguments to specify binary location."
    exit 1
fi


# Build filestoragevm, which is run as a subprocess
echo "Building filestoragevm in $binary_directory/$name"
go build -o "$binary_directory/$name" "main/"*.go
