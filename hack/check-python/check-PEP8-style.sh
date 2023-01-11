#!/usr/bin/env bash

. ./hack/check-python/prepare-env.sh

echo "----------------------------------------------------"
echo "Running Python linter against"
echo "the following directories:"
echo "$directories"
echo "----------------------------------------------------"
echo

[ "$NOVENV" == "1" ] || prepare_venv || exit 1

# checks for the whole directories
for directory in $directories
do
    files=$(find "$directory" -path "$PYTHON_VENV_DIR" -prune -o -name '*.py' -print)

    for source in $files
    do
        echo "$source"
        "$PYTHON_VENV_DIR/bin/pycodestyle" "$source"
        exit=$?
        if [ $exit -eq 0 ]
        then
            echo "    Pass"
            (( pass++ ))
        else
            echo "    Fail"
            (( fail++ ))
        fi
    done
done


if [ "$fail" -eq 0 ]
then
    echo "All checks passed for $pass source files"
else
    (( total=pass+fail ))
    echo "Linter fail, $fail source files out of $total source files need to be fixed"
    exit 1
fi
