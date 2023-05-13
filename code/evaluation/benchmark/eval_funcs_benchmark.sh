function run_benchmark_download_known_file {
    # All Swarm pods: Restart and restore from backup.
    restartAndRestorePodsFromBackup ${7}

    # Test download with known file
    test_download_snarl "${1}" "${2}" "${3}" "${4}" "${5}" "${6}" "${7}"
}

# Params: ITERATIONS SIZE NUMPEERS FAILEDNODES FAILRATE REPLICATION
function run_benchmark_download_random_file {
    # All Swarm pods: Restart and restore from backup.
    restartAndRestorePodsFromBackup ${6}

    # Test download with random file
    generate_and_upload_testfile "${2}" "false" "true"
    test_download_snarl "${1}" $HASHIDS $RANDOMSHASUM "${3}" "${4}" "${5}" "${6}"
}

# Param1: Variable identifier
function set_variables {
    case ${1} in
        1MB) source eval_vars/1MB.sh ;;
        5MB) source eval_vars/5MB.sh ;;
        *) echo "Invalid identifier." ;;
    esac
}