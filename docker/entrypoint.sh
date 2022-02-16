#!/bin/bash

# Start the run once job.
echo "Docker container has been started"

# PostgreSQL run
/usr/lib/postgresql/"$PGDBVERSION"/bin/postgres -D /var/lib/postgresql/"$PGDBVERSION"/main -c config_file=/etc/postgresql/"$PGDBVERSION"/main/postgresql.conf