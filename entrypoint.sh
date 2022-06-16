#!/bin/sh

set -x

/app/main migrate init
/app/main migrate
/app/main serve

# exec "$@"
