#!/bin/sh
set -e

/app/main migrate
/app/main serve

exec "$@"
