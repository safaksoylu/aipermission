#!/bin/sh
set -e

mkdir -p /app/data
chown -R aipermission:nogroup /app/data

exec gosu aipermission "$@"
