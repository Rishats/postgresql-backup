#!/bin/bash

go mod download

go build -o postgresql-backup

chmod +x postgresql-backup

sudo ./postgresql-backup