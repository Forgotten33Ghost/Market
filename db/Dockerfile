FROM postgres:15

COPY ./dump.sql /docker-entrypoint-initdb.d/dump.sql
COPY ./init.sh /docker-entrypoint-initdb.d/init.sh
RUN chmod +x /docker-entrypoint-initdb.d/init.sh