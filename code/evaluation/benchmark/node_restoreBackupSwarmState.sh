function restoreBackupSwarmState {
    if [[ ${1} == "backup" ]]; then
        # Do nothing. (Yet ... :) )
        echo "a"
    elif [[ ${1} == "quickrestore" ]]; then
        if [[ ! -z ${2} ]]; then
            # Replication parameter was passed in.
            local lookForDir="-"${2}
        else
            local lookForDir="-backup"
        fi
        # Find all directories that we want to copy from.
        local allDirs=$(ls -d /mnt/local-storage/*)
        for dir in $allDirs
        do
            # We are only interested in backup directories.
            if [[ $dir != *$lookForDir ]]; then
                #echo "$dir does not end with $lookForDir"
                continue
            fi

            # Find original directory
            local orgDir=$(echo ${dir} | awk -F ${lookForDir} '{print $1}')

            # Remove contents of original directory
            rm -rf ${orgDir}/*

            # Copy from backup to original
            cp -R ${dir}/. ${orgDir}/

            echo "rm -rf ${orgDir}/*"
            echo "cp -R ${dir}/. ${orgDir}/"
        done
    else
        echo "Pass one of these parameters: backup | restore | quickrestore"
    fi
}

restoreBackupSwarmState "${1}" "${2}"