#!/bin/bash

__DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

function run_all_tests {
    command="" 
    for dir in test/*/*; do
        if [[ $dir != *"util"* ]]; then {
            command+=" "$dir
        } 
        fi
    done
    ginkgo -v $command
}

function run_feature_tests {
    for feature in ${FEATURES}
    do
        for dir in test/*/*; do
        if [[ $dir != *"util"* ]] && [[ $dir == *"${feature}"* ]]; then {
            command+=" "$dir
        } fi
        done
    done
    ginkgo -v $command
}

function argument_sorter {
    case $1 in
        all)
            echo "#### Run all tests ####"
            run_all_tests
            ;;
        features)
            if [ -z "$FEATURES" ]; then {
                echo "FEATURES env var is empty. Please export FEATURES"
                exit 1
            } fi
            echo "#### Run feature tests: ${FEATURES} ####"
            run_feature_tests ${FEATURES}
            ;;
        *)
        exit 1
        ;;
    esac
}

argument_sorter ${1}
