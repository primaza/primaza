#!/usr/bin/env bash

directories=${directories:-"test/acceptance/features"}
export pass=0
export fail=0

export PYTHON_VENV_DIR=${PYTHON_VENV_DIR:-venv}

function prepare_venv() {
    # shellcheck disable=SC1090 disable=SC1091
    python3 -m venv "$PYTHON_VENV_DIR" && source "$PYTHON_VENV_DIR/bin/activate"
    while IFS= read -r -d '' req; do
        python3 "$(command -v pip3)" install -q -r "$req"
    done < <(find . -name 'requirements.txt' -print0)
    python3 "$(command -v pip3)" install -q -r "$(dirname "$0")/requirements.txt"
}
