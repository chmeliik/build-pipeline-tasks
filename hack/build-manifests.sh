#!/usr/bin/env bash
set -e

# <TEMPLATED FILE!>
# This file comes from the templates at https://github.com/konflux-ci/task-repo-shared-ci.
# Please consider sending a PR upstream instead of editing the file directly.
# See the SHARED-CI.md document in this repo for more details.

# To make the script work on linux and mac, use '${SED_CMD}' instead of 'sed'
# https://stackoverflow.com/a/4247319
if [[ "$OSTYPE" == "darwin"* ]]; then
  # Require gnu-sed.
  if ! [ -x "$(command -v gsed)" ]; then
    echo "Error: 'gsed' is not installed." >&2
    echo "If you are using Homebrew, install with 'brew install gnu-sed'." >&2
    exit 1
  fi
  SED_CMD="gsed"
else
  SED_CMD="sed"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"

# You can ignore building manifests for some tasks by providing the SKIP_TASKS variable
# with the task name separated by a space, for example:
# SKIP_TASKS="git-clone init"

SKIP_TASKS=

: "${ROOT_MANIFESTS_DIR=.}"

warning_message="# WARNING: This is an auto generated file, do not modify this file directly"

main() {
    cd "$SCRIPT_DIR/.."
    local ret=0
    while read -r kustomization_yaml_path
    do
        # Strip leading ./
        kustomization_yaml_path=${kustomization_yaml_path#./}

        resource_name=$(echo "$kustomization_yaml_path" | cut -d'/' -f2)
        resource_dir=$(dirname "$kustomization_yaml_path")

        echo "Building manifest for $resource_dir"

        # Skip the resources mentioned in SKIP_TASKS
        skipit=
        for tname in ${SKIP_TASKS};do
            [[ ${tname} == "${resource_name}" ]] && skipit=True
        done
        [[ -n ${skipit} ]] && continue

        # Check if there is only one resource in the kustomization file and it is <resource_name>.yaml
        resources=$(yq -r '.resources[]' "$kustomization_yaml_path")
        if [[ "$resources" == "$resource_name.yaml" ]]; then
          echo "Skip generating manifest for $resource_dir"
          continue
        fi
        if ! oc kustomize -o "$resource_dir/$resource_name.yaml" "$resource_dir/"; then
            echo "failed to build $resource_name" >&2
            ret=1
            continue
        fi
        # Add a warning message in the generated file
        ${SED_CMD} -i "1 i $warning_message" "$resource_dir/$resource_name.yaml"
    done < <(find "$ROOT_MANIFESTS_DIR" -type f -name "kustomization.yaml")

    exit "$ret"

}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
