#!/bin/sh
set -eu

data_dir="${ADASH_DB_PATH%/*}"
if [ -z "$data_dir" ] || [ "$data_dir" = "$ADASH_DB_PATH" ]; then
  data_dir=/app/data
fi

case "$data_dir" in
  /app/data|/app/data/*) ;;
  *)
    echo "HATA: ADASH_DB_PATH yalnızca /app/data altında olabilir: $ADASH_DB_PATH" >&2
    exit 1
    ;;
esac

mkdir -p "$data_dir"
chown -R adash:adash /app/data
exec gosu adash "$@"
