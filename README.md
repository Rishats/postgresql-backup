# PostgreSQL Backup

Simple system which use pg_dump for dump postgresql db and send info in to Telegram via [Horn](https://github.com/requilence/integram)

### Installing for develop purpose
1) Clone project
    ```bash
    git clone https://github.com/Rishats/postgresql-backup.git
    ```
2) Change folder
    ```bash
    cd postgresql-backup
    ```
3) Create .env file from .env.example
    ```bash
     cp .env.example .env
    ```

4) Configure your .env
    ```
       APP_ENV=production-or-other
       POSTGRESQL_HOST=127.0.0.1(or-empty-if-localhost)
       POSTGRESQL_PORT=3306
       POSTGRESQL_DB=mydb
       POSTGRESQL_USER=mydbuser
       BACKUP_DIR=/var/lib/postgresql/backups/
       INTEGRAM_WEBHOOK_URI=your-uri
       SENTRY_DSN=your-dsn
       ROTATED_TIME_DAILY=7
       ROTATED_TIME_WEEKLY=4(or-empty-if-no-need)
       ROTATED_TIME_MONTHLY=12(or-empty-if-no-need)
    ```

### Develop use cases

#### Via go native:

Download dependency
```bash
go mod download
```

Build for linux
```bash
env GOOS=linux GOARCH=amd64 go build main.go
```

#### Via docker:
```bash
 docker build --target=build-env -t postgresql-backup .
 docker run -d --name "postgresql-backup" postgresql-backup
```


### Usage

#### Via Ansible playbooks
https://github.com/Rishats/ansible-postgresql-backup

#### Via simple crontab
1) Create scripts folder and download latest release with custom .env
   ```bash
   mkdir scripts
   cp .env.example .env
   vim .env
   ```
2) Added example crontab entry
   
    ```bash
    crontab -u postgres -e
    ```

    ```bash
    0 2 * * * /var/lib/postgresql/scripts/postgresql-backup > /dev/null 2>&1
    ```

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/Rishats/ywpti/tags). 

## Authors

* **Rishat Sultanov** - [Rishats](https://github.com/Rishats)

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
