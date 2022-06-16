#!/bin/sh

set -x

/app/main init
/app/main migrate
/app/main serve

# exec "$@"
