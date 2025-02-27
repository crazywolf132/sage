#!/bin/bash
set -e

echo "Testing tag retrieval and version calculation"
echo "--------------------------------------------"

# Check if repo is shallow before using --unshallow
if [ -f .git/shallow ]; then
  echo "Repository is shallow, using --unshallow flag"
  git fetch --prune --unshallow --tags
else
  echo "Repository is not shallow, skipping --unshallow flag"
  git fetch --prune --tags
fi

# Sort tags by version number and get the latest
latest_tag=$(git tag -l "v*" | sort -V | tail -n 1)

if [ -z "$latest_tag" ]; then
  echo "No tags found, would start from v0.1.0"
  commits=$(git log --pretty=format:"%s")
  major=0
  minor=1
  patch=0
  is_initial_release=true
else
  echo "Latest tag: $latest_tag"
  commits=$(git log --pretty=format:"%s" ${latest_tag}..HEAD)
  version=${latest_tag#v}
  IFS='.' read -r major minor patch <<< "$version"
  is_initial_release=false
fi

echo ""
echo "Analyzing commits since ${latest_tag:-beginning}..."
echo "Total commits to analyze: $(echo "$commits" | wc -l | xargs)"

# Print a sample of commits (up to 5)
echo ""
echo "Sample of commits being analyzed:"
echo "$commits" | head -n 5

echo ""
echo "Commit classification:"
# Count meaningful changes with precise patterns
major_bump=$(echo "$commits" | grep -cE "^(feat|fix|refactor|perf)!:" || true)
minor_bump=$(echo "$commits" | grep -cE "^(feat|refactor)(\([^)]+\))?:" || true)
patch_bump=$(echo "$commits" | grep -cE "^fix(\([^)]+\))?:" || true)

# Calculate total meaningful changes
meaningful_changes=$((major_bump + minor_bump + patch_bump))

echo "Major changes (breaking): $major_bump"
echo "Minor changes (features): $minor_bump"
echo "Patch changes (fixes): $patch_bump"
echo "Total meaningful changes: $meaningful_changes"

# Calculate next version
echo ""
echo "Version calculation:"
echo "Current version: ${latest_tag:-none}"

if [ "$is_initial_release" = "true" ] || [ $meaningful_changes -gt 0 ]; then
  if [ "$is_initial_release" = "true" ]; then
    echo "This would be the initial release"
  elif [ $major_bump -gt 0 ]; then
    echo "Major version bump needed"
    major=$((major + 1))
    minor=0
    patch=0
  elif [ $minor_bump -gt 0 ]; then
    echo "Minor version bump needed"
    minor=$((minor + 1))
    patch=0
  elif [ $patch_bump -gt 0 ]; then
    echo "Patch version bump needed"
    patch=$((patch + 1))
  fi
  
  new_tag="v${major}.${minor}.${patch}"
  echo "Next version would be: $new_tag"
else
  echo "No meaningful changes detected, no new version would be created"
fi

echo ""
echo "Test completed successfully!" 