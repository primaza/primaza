#!/bin/bash -e

TEST_ACCEPTANCE_FEATURES_DIR=${TEST_ACCEPTANCE_FEATURES_DIR:-$(dirname "$0")/../test/acceptance/features}

FORBIDDEN_TAGS="@wip @dev"

failed=""

for t in $FORBIDDEN_TAGS; do
    echo "----------------------------------------------------------"
    echo "Checking for forbidden tag '$t' in feature files:"
    echo "----------------------------------------------------------"
    while IFS= read -r -d '' file; do
        echo -e "\nChecking $file"
        match=$(grep -PirnH "$t" "$file") || true
        if [[ -z $match ]]; then
            echo -n "    PASS"
        else
            echo -en "    FAIL\n$match\n"
            failed="$failed\n$match"
        fi
    done < <(find "$TEST_ACCEPTANCE_FEATURES_DIR" -name '*.feature' -print0)
    echo
    echo
done

if [ -n "$failed" ]; then
    echo -e "\nERROR: Following feature file checks FAILED:$failed\n"
    exit 1
else
    echo -e "\nAll feature file checks PASSED\n"
fi
