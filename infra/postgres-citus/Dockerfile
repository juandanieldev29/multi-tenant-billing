FROM citusdata/citus:12.1

# Init scripts run in alphabetical order on first container start.
# They are skipped on subsequent starts (data directory already initialized).
COPY init/ /docker-entrypoint-initdb.d/

# Tune PostgreSQL for OLAP-style aggregation queries and write-heavy ingestion.
# These are appended to postgresql.conf by the Citus entrypoint.
ENV POSTGRES_SHARED_PRELOAD_LIBRARIES=citus
ENV PGDATA=/var/lib/postgresql/data/pgdata

RUN echo "shared_buffers = 512MB"           >> /etc/postgresql/postgresql.conf.sample && \
    echo "work_mem = 64MB"                  >> /etc/postgresql/postgresql.conf.sample && \
    echo "maintenance_work_mem = 256MB"     >> /etc/postgresql/postgresql.conf.sample && \
    echo "max_connections = 200"            >> /etc/postgresql/postgresql.conf.sample && \
    echo "wal_level = logical"              >> /etc/postgresql/postgresql.conf.sample && \
    echo "max_wal_senders = 10"             >> /etc/postgresql/postgresql.conf.sample && \
    echo "checkpoint_completion_target = 0.9" >> /etc/postgresql/postgresql.conf.sample && \
    echo "default_statistics_target = 200"  >> /etc/postgresql/postgresql.conf.sample && \
    echo "random_page_cost = 1.1"           >> /etc/postgresql/postgresql.conf.sample

EXPOSE 5432
