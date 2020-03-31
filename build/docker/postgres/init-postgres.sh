#!/bin/sh
set -e

mkdir -p /var/lib/postgresql/data/pgdata/
cp /etc/postgresql/postgresql.conf /var/lib/postgresql/data/pgdata/postgresql.conf

psql -v ON_ERROR_STOP=1 --username ${POSTGRES_USER} --dbname ${POSTGRES_DB} <<-EOSQL
    CREATE USER pipes_user;
    ALTER USER pipes_user SUPERUSER;
EOSQL

psql ${POSTGRES_USER} -c 'select pg_reload_conf()'

psql -c 'DROP DATABASE IF EXISTS pipes_test;' -U ${POSTGRES_USER}
psql -c 'CREATE DATABASE pipes_test;' -U ${POSTGRES_USER}
psql pipes_test < /schema.sql

psql -c 'DROP DATABASE IF EXISTS pipes_development;' -U ${POSTGRES_USER}
psql -c 'CREATE DATABASE pipes_development;' -U ${POSTGRES_USER}
psql pipes_development < /schema.sql
