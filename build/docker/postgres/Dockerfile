FROM postgres:9.6
MAINTAINER anton.kucherov@toggl.com

ENV POSTGRES_PASSWORD=""

ADD postgres.conf /etc/postgresql/postgresql.conf
ADD init-postgres.sh /docker-entrypoint-initdb.d/init-postgres.sh
ADD schema.sql /schema.sql
