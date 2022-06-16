#!/bin/sh

set -x

env

/app/main migrate
/app/main serve

# exec "$@"
