#!/bin/sh

set -x

/app/main migrate init
/app/main serve

# exec "$@"
