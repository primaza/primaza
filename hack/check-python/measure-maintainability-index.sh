#!/usr/bin/env bash

. ./hack/check-python/prepare-env.sh

[ "$NOVENV" == "1" ] || prepare_venv || exit 1

SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"

echo "----------------------------------------------------"
echo "Checking source files for maintainability index"
echo "in the following directories:"
echo "$directories"
echo "----------------------------------------------------"
echo
for directory in ${SCRIPT_DIR}/.. $directories; do
    pushd "$directory" || exit
    "$PYTHON_VENV_DIR/bin/radon" mi -s -i "$PYTHON_VENV_DIR" .
    popd || exit
done
