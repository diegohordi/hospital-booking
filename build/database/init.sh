#!/bin/bash
set -e

export PGPASSWORD=$POSTGRES_PASSWORD;
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
  CREATE USER $APP_USER WITH PASSWORD '$APP_PASS';
  CREATE DATABASE $APP_DB;
  GRANT ALL PRIVILEGES ON DATABASE $APP_DB TO $APP_USER;
EOSQL

psql -v ON_ERROR_STOP=1 --username "$APP_USER" --dbname "$APP_DB" -a -f /tmp/hospital_booking.sql