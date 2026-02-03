#!/usr/bin/env sh
set -e
if command -v openssl >/dev/null 2>&1; then
  openssl rand -base64 32
else
  head -c 32 /dev/urandom | base64
fi
