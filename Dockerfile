FROM golang:1.17.1 AS build-env
ENV GO111MODULE=on
WORKDIR /app/postgresql-backup
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o postgresql-backup

FROM ubuntu:20.04 AS dev-build
ENV TZ="Asia/Almaty"
ENV DEBIAN_FRONTEND=noninteractive
ENV PGDBVERSION=12
ENV PGPASSWORD='docker'

RUN apt -y update
RUN apt -y install postgresql postgresql-contrib postgresql-client vim ca-certificates && update-ca-certificates

# Run the rest of the commands as the ``postgres`` user created by the ``postgres`` package when it was ``apt-get installed``
USER postgres

# Adjust PostgreSQL configuration so that remote connections to the
# database are possible.
RUN echo "host all  all    0.0.0.0/0  md5" >> /etc/postgresql/"$PGDBVERSION"/main/pg_hba.conf

# And add ``listen_addresses`` to ``/etc/postgresql/12/main/postgresql.conf``
RUN echo "listen_addresses='*'" >> /etc/postgresql/"$PGDBVERSION"/main/postgresql.conf

# Create a PostgreSQL role named ``docker`` with ``docker`` as the password and
# then create a database `docker` owned by the ``docker`` role.
# Note: here we use ``&&\`` to run commands one after the other - the ``\``
#       allows the RUN command to span multiple lines.
RUN /etc/init.d/postgresql start &&\
    psql --command "CREATE USER docker WITH SUPERUSER PASSWORD 'docker';" &&\
    createdb -O docker docker &&\
    psql -h 127.0.0.1 -U docker docker --command "CREATE TABLE accounts (username VARCHAR ( 50 ) UNIQUE NOT NULL);" &&\
    psql -h 127.0.0.1 -U docker docker --command "INSERT INTO accounts(username) VALUES ('rishatsultanov');"

# Expose the PostgreSQL port
EXPOSE 5432

# Add VOLUMEs to allow backup of config, logs and databases
VOLUME  ["/etc/postgresql", "/var/log/postgresql", "/var/lib/postgresql"]

# BACKUP script rotate.
WORKDIR /var/lib/postgresql/scripts
COPY --from=build-env /app/postgresql-backup/postgresql-backup .
COPY --from=build-env /app/postgresql-backup/.env.example .env
USER root
RUN chmod +x postgresql-backup

# Add files
ADD docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

USER postgres
ENTRYPOINT /entrypoint.sh