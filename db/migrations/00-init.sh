#!/bin/bash
set -e

cd /docker-entrypoint-initdb.d
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -f 00001.create_base.sql
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -f 00002.exchange_rates.sql 