# PostgreSQL Backup

Simple system which use pg_dump for dump postgresql db and send info in to Telegram via [Horn](https://github.com/requilence/integram)

### Installing
1) Clone project
    ```
    git clone https://github.com/Rishats/postgresql-backup.git
    ```
2) Change folder
    ```
    cd postgresql-backup
    ```
3) Create .env file from .env.example
    ```
     cp .env.example .env
    ```

4) Configure your .env
    ```
       APP_ENV=production-or-other
       POSTGRESQL_HOST=127.0.0.1
       POSTGRESQL_PORT=3306
       POSTGRESQL_DB=mydb
       POSTGRESQL_USER=mydbuser
       BACKUP_DIR=/var/lib/postgresql/backups
       INTEGRAM_WEBHOOK_URI=your-uri
       SENTRY_DSN=your-dsn
    ```

### Running

#### Via go native:

Download dependency
```
go mod download
```

Build for linux
```
env GOOS=linux GOARCH=amd64 go build main.go
```

#### Via docker:

```
 docker build -t postgresql-backup .
 docker run -d --name "postgresql-backup" postgresql-backup
```

## Versioning

We use [SemVer](http://semver.org/) for versioning. For the versions available, see the [tags on this repository](https://github.com/Rishats/ywpti/tags). 

## Authors

* **Rishat Sultanov** - [Rishats](https://github.com/Rishats)

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
