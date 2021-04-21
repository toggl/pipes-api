#!/usr/bin/env bash

set -euo pipefail

mkdir -p tmp
echo 'Fetching config...'
[[ -d config ]] || git clone git@github.com:toggl/pipes-api-conf.git config

cd config
git fetch
git checkout master
git reset --hard origin/master
