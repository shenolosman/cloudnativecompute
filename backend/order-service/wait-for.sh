#!/bin/sh
set -e

host="$1"
shift
cmd="$@"

until nc -z "$host" 2>/dev/null; do
  echo "Waiting for $host..."
  sleep 1
done

exec $cmd
