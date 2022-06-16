#!/bin/sh

set -x

/app/main migrate
/app/main serve

# exec "$@"
