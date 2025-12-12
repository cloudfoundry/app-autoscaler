#! /usr/bin/env bash
# shellcheck disable=SC2154,SC1091

set -euo pipefail
script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
autoscaler_dir="${script_dir}/.."

# Source common functions
source "${script_dir}/common.sh"

mkdir -p "build"
build_path=$(realpath build)
mkdir -p "keys"
keys_path="$(realpath keys)"
PROMOTE_DRAFT=${PROMOTE_DRAFT:-"false"}
REPO_OUT=${REPO_OUT:-}
SUM_FILE="${build_path}/artifacts/files.sum.sha256"

function bump_version() {
   set -e
   local version=$1
   echo " - bumping version to ${version} in API files at revision $(git rev-parse HEAD)"
   yq eval -i ".build = \"${version}\"" api/default_info.json

   git add api/default_info.json
   git commit -S -m "Updated release version to ${version} in golangapiserver"
}

function generate_changelog(){
  [ -e "${build_path}/changelog.md" ] && return
  echo " - Generating changelog using github cli..."
  mkdir -p "${build_path}"

  # If promoting an existing draft, find the latest draft release
  if [ "${PROMOTE_DRAFT}" == "true" ]; then
    echo " - Looking for latest draft release..."
    local draft_version
    draft_version=$(gh release list --limit 10 --json tagName,isDraft --jq '.[] | select(.isDraft == true) | .tagName' | head -1)

    if [ -z "$draft_version" ]; then
      echo " - ERROR: No draft release found to promote"
      exit 1
    fi

    # Update VERSION to match the draft we found (strip "v" prefix if present)
    VERSION="${draft_version#v}"
    echo " - Found draft release v${VERSION}, will promote to final"
    echo "${VERSION}" > "${build_path}/name"
    gh release view "v${VERSION}" --json body --jq '.body' > "${build_path}/changelog.md"
  else
    # Otherwise, create a new draft release (or recreate if draft exists)
    # First delete any existing draft releases with matching version (handles untagged drafts)
    echo " - Checking for existing draft releases with version ${VERSION}..."
    local existing_drafts
    existing_drafts=$(gh release list --limit 20 --json tagName,name,isDraft --jq ".[] | select(.isDraft == true and (.tagName == \"v${VERSION}\" or .name == \"v${VERSION}\" or .name == \"${VERSION}\")) | .tagName")

    if [ -n "$existing_drafts" ]; then
      while IFS= read -r draft_tag; do
        if [ -n "$draft_tag" ]; then
          echo " - Deleting existing draft release: ${draft_tag}"
          gh release delete "${draft_tag}" --yes --cleanup-tag || true
        fi
      done <<< "$existing_drafts"
    fi

    # Check if there's a published release with this version
    if gh release view "v${VERSION}" &>/dev/null; then
      local is_draft
      is_draft=$(gh release view "v${VERSION}" --json isDraft --jq '.isDraft')
      if [ "$is_draft" = "false" ]; then
        echo " - ERROR: Release v${VERSION} already exists and is published (not a draft)"
        echo " - Refusing to delete published release. Please check version logic."
        exit 1
      fi
    fi

    echo " - Creating new draft release v${VERSION}..."
    gh release create "v${VERSION}" --generate-notes --draft
    gh release view "v${VERSION}" --json body --jq '.body' > "${build_path}/changelog.md"
  fi
}

function setup_git(){
	echo " - Setting up git for signing commits..."
  if [[ -z $(git config --global user.email) ]]; then
		echo " - Configuring git user email ${AUTOSCALER_CI_BOT_EMAIL}..."
    git config --global user.email "${AUTOSCALER_CI_BOT_EMAIL}"
  fi

  if [[ -z $(git config --global user.name) ]]; then
		echo " - Configuring git user name ${AUTOSCALER_CI_BOT_NAME}..."
    git config --global user.name "${AUTOSCALER_CI_BOT_NAME}"
  fi

  public_key_path="${keys_path}/autoscaler-ci-bot-signing-key.pub"
  private_key_path="${keys_path}/autoscaler-ci-bot-signing-key"
  echo "$AUTOSCALER_CI_BOT_SIGNING_KEY_PUBLIC" > "${public_key_path}"
  echo "$AUTOSCALER_CI_BOT_SIGNING_KEY_PRIVATE" > "${private_key_path}"
  chmod 600 "${public_key_path}"
  chmod 600 "${private_key_path}"

  git config --global gpg.format ssh
  git config --global user.signingkey "${private_key_path}"
}


pushd "${autoscaler_dir}" > /dev/null
  determine_next_version

  VERSION=${VERSION:-$(cat "${build_path}/name")}
  generate_changelog

  echo " - Displaying diff..."
  export GIT_PAGER=cat
  git diff
  echo "v${VERSION}" > "${build_path}/tag"

  # Build artifacts only when promoting a draft to final
  if [ "${PROMOTE_DRAFT}" == "true" ]; then
		setup_git
    bump_version "${VERSION}"

    # Update the tag to point to the new commit created by bump_version
    echo " - Updating tag v${VERSION} to point to current commit $(git rev-parse HEAD)"
    git tag -f "v${VERSION}"

    ACCEPTANCE_TEST_TGZ="app-autoscaler-acceptance-tests-v${VERSION}.tgz"
    AUTOSCALER_MTAR="app-autoscaler-release-v${VERSION}.mtar"

    # Extract SHA256 checksums from files created by create-assets target
    ACCEPTANCE_SHA256=$( grep "${ACCEPTANCE_TEST_TGZ}$" "${SUM_FILE}" | awk '{print $1}' )
    MTAR_SHA256=$( grep "${AUTOSCALER_MTAR}$" "${SUM_FILE}" | awk '{print $1}')
  else
    ACCEPTANCE_SHA256="dummy-sha"
    MTAR_SHA256="dummy-sha"
  fi
  export ACCEPTANCE_SHA256
  export MTAR_SHA256

  cat >> "${build_path}/changelog.md" <<EOF

## Deployment

\`\`\`yaml
releases:
- name: app-autoscaler-acceptance-tests
  sha1: sha256:${ACCEPTANCE_SHA256}
- name: app-autoscaler-mtar
  sha1: sha256:${MTAR_SHA256}
\`\`\`
EOF

	echo "find changelog at https://github.com/cloudfoundry/app-autoscaler-release/releases/tag/v${VERSION}"

  # If promoting draft to final, upload artifacts and publish
  if [ "${PROMOTE_DRAFT}" == "true" ]; then
    echo " - Uploading artifacts to release v${VERSION}..."
    gh release upload "v${VERSION}" "${build_path}/artifacts/"* --clobber

    echo " - Updating release notes with deployment information..."
    gh release edit "v${VERSION}" --notes-file "${build_path}/changelog.md"

    echo " - Publishing release v${VERSION}..."
    gh release edit "v${VERSION}" --draft=false
    echo " - Release v${VERSION} published successfully!"
  fi

popd > /dev/null
echo " - Completed"
