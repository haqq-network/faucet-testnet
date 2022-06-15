#!/bin/sh
set -e

./main migrate
./main serve

exec "$@"