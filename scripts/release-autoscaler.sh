#! /usr/bin/env bash
# shellcheck disable=SC2154,SC1091

set -euo pipefail
script_dir=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
source "${script_dir}/vars.source.sh"

previous_version=${PREV_VERSION:-$(gh release list --limit 1 --exclude-drafts --exclude-pre-releases --json tagName --jq '.[0].tagName' 2>/dev/null)}
# If no previous version found, default to v15.9.0
if [ -z "$previous_version" ] || [ "$previous_version" = "null" ]; then
  previous_version="v15.9.0"
fi
echo " - Previous version from GitHub releases: ${previous_version}"

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

function create_mtar() {
  set -e
  mkdir -p "${build_path}/artifacts"
  local version=$1
  local build_path=$2
  echo " - creating autorscaler mtar artifact"
  pushd "${autoscaler_dir}" > /dev/null
    make mta-release VERSION="${version}" DEST="${build_path}/artifacts/"
  popd > /dev/null
}

function create_tests() {
  set -e
  mkdir -p "${build_path}/artifacts"
  local version=$1
  local build_path=$2
  echo " - creating acceptance test artifact"
  pushd "${autoscaler_dir}" > /dev/null
    make acceptance-release VERSION="${version}" DEST="${build_path}/artifacts/"
  popd > /dev/null
}

function determine_next_version(){
  echo " - Determining next version..."

  # Check if there's an existing draft release
  local draft_version
  draft_version=$(gh release list --limit 10 --json tagName,isDraft --jq '.[] | select(.isDraft == true) | .tagName' | head -1)

  if [ -n "$draft_version" ]; then
    echo " - Found existing draft release: ${draft_version}"
    echo " - Using draft version as next version"
    echo "${draft_version#v}" > "${build_path}/name"
    return
  fi

  # If no draft found, continue with version calculation
  echo " - No draft release found, calculating version from commits..."
  echo " - Previous version: $previous_version"

  # Remove 'v' prefix if present
  local version_number=${previous_version#v}

  # Parse version components
  IFS='.' read -r major minor patch <<< "$version_number"

  # Get commits since last tag
  local commits_since_tag
  commits_since_tag=$(git rev-list "${previous_version}"..HEAD --oneline 2>/dev/null || git rev-list HEAD --oneline)
  local commit_count
  commit_count=$(echo "$commits_since_tag" | wc -l)

  if [ -z "$commits_since_tag" ] || [ "$commit_count" -eq 0 ]; then
    echo " - No commits since last tag, keeping current version"
    echo "$version_number" > "${build_path}/name"
    return
  fi

  # Extract PR numbers from commits (supports both "(#123)" and " #123 " formats)
  local pr_numbers
  pr_numbers=$(echo "$commits_since_tag" | grep -oE '(\(#[0-9]+\)| #[0-9]+ )' | grep -oE '[0-9]+' | sort -u)

  if [ -z "$pr_numbers" ]; then
    echo " - No PR numbers found in commits, incrementing patch version"
    patch=$((patch + 1))
    local new_version="${major}.${minor}.${patch}"
    echo " - Next version: $new_version"
    echo "$new_version" > "${build_path}/name"
    return
  fi

  # Query GitHub API for PR labels and categorize
  local has_breaking=0
  local has_enhancement=0
  local pr_count=0

  echo " - Checking PR labels for version determination..."
  while IFS= read -r pr_num; do
    if [ -n "$pr_num" ]; then
      pr_count=$((pr_count + 1))
      local labels
      labels=$(gh pr view "$pr_num" --json labels --jq '.labels[].name' 2>/dev/null || echo "")

      if echo "$labels" | grep -q "exclude-from-changelog"; then
        echo "   - PR #$pr_num: excluded from changelog"
        continue
      fi

      if echo "$labels" | grep -q "breaking-change"; then
        echo "   - PR #$pr_num: breaking change"
        has_breaking=1
      elif echo "$labels" | grep -q "enhancement"; then
        echo "   - PR #$pr_num: enhancement"
        has_enhancement=1
      fi
    fi
  done <<< "$pr_numbers"

  # Determine version increment based on PR labels
  if [[ "$has_breaking" -eq 1 ]]; then
    major=$((major + 1))
    minor=0
    patch=0
    echo " - Found breaking changes, incrementing major version"
  elif [[ "$has_enhancement" -eq 1 ]]; then
    minor=$((minor + 1))
    patch=0
    echo " - Found enhancements, incrementing minor version"
  else
    patch=$((patch + 1))
    echo " - Found changes, incrementing patch version"
  fi

  local new_version="${major}.${minor}.${patch}"
  echo " - Next version: $new_version"
  echo "$new_version" > "${build_path}/name"
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
    return
  fi

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
}

function setup_git(){
  if [[ -z $(git config --global user.email) ]]; then
    git config --global user.email "${AUTOSCALER_CI_BOT_EMAIL}"
  fi

  if [[ -z $(git config --global user.name) ]]; then
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
    ACCEPTANCE_TEST_TGZ="app-autoscaler-acceptance-tests-v${VERSION}.tgz"
    AUTOSCALER_MTAR="app-autoscaler-release-v${VERSION}.mtar"

    mkdir -p "${build_path}/artifacts"
    create_tests "${VERSION}" "${build_path}"
    create_mtar "${VERSION}" "${build_path}"

    sha256sum "${build_path}/artifacts/"* > "${build_path}/artifacts/files.sum.sha256"
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
  version: ${VERSION}
  url: https://storage.googleapis.com/app-autoscaler-releases/releases/app-autoscaler-acceptance-tests-v${VERSION}.tgz
  sha1: sha256:${ACCEPTANCE_SHA256}
- name: app-autoscaler-mtar
  version: ${VERSION}
  url: https://storage.googleapis.com/app-autoscaler-releases/releases/app-autoscaler-release-v${VERSION}.mtar
  sha1: sha256:${MTAR_SHA256}
\`\`\`
EOF
  echo "---------- Changelog file ----------"
  cat "${build_path}/changelog.md"
  echo "---------- end file ----------"

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
