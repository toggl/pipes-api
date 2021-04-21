#!/usr/bin/env bash

set -euo pipefail

mkdir -p tmp
echo 'Fetching deploy-script...'
[[ -d tmp/deploy-script ]] || git clone git@github.com:toggl/deploy-script.git tmp/deploy-script

cd tmp/deploy-script
git fetch
git checkout master
git reset --hard origin/master
