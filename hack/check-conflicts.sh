#!/usr/bin/env bash

echo -e "\nChecking for presence of conflict notes in source files..."

check_file(){
    grep -PiIrnH '<<<<''<<<|>>>''>>>>|===''====' "$1"
    if [ $? -ne 1 ]; then
        echo -e "$1:\n\tFAIL"
        return 1
    else
        echo -e "$1:\n\tPASS"
        return 0
    fi
}

overall_failed=0

echo -e "\nChecking staged files... "
while IFS='' read -r file; do
    # check_file $file || overall_failed=1
    check_file "$file" || ((overall_failed++))
done < <(git diff --cached --name-only | grep -v "vendor/")

echo -e "\nChecking tracked files... "
while IFS='' read -r file; do
    check_file "$file" || ((overall_failed++))
done < <(git ls-tree --full-tree -r HEAD --name-only | grep -v "vendor/" )

if [ $overall_failed -eq 0 ]; then
    echo -e "\nNone of the tracked or staged files contain strings indicating a git conflict... PASS\n"
else
    echo -e "\nThe above listed files contains strings indicating a git conflict... FAIL\n"
fi
exit $overall_failed
