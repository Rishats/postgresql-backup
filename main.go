package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/getsentry/raven-go"
	"github.com/h2non/filetype"
	"github.com/ivahaev/russian-time"
	"github.com/jasonlvhit/gocron"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func sendToHorn(text string) {
	m := map[string]interface{}{
		"text": text,
	}
	mJson, _ := json.Marshal(m)
	contentReader := bytes.NewReader(mJson)
	req, err := http.NewRequest("POST", os.Getenv("INTEGRAM_WEBHOOK_URI"), contentReader)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Panic(err)
	}

	fmt.Println(resp)
}

func hourWithMin() string {

	timeStamp := time.Unix(time.Now().Unix(), 0)

	hr, min, _ := timeStamp.Clock()

	finalTime := "%d:%d"

	result := fmt.Sprintf(finalTime, hr, min)

	return result
}

func weekDay() rtime.Weekday {
	t := rtime.Now()
	standardTime := time.Now()
	t = rtime.Time(standardTime)

	return t.Weekday()
}

func cleanerSuccess(fileName string) {
	sendToHorn(fmt.Sprintf("[PostgreSQL 📦 - 👴🏿] Старый бэкап [%s] был успешно удален! ✅", fileName))
}

func fileNameGenerate() string {
	currentTime := time.Now()
	filename := os.Getenv("POSTGRESQL_DB") + "-" + currentTime.Format("2006_01_02_15_04_05") + ".sql.gz"
	return filename

}

func generatePostgresqlDumpOptions(fileName string) string {
	var options string

	if os.Getenv("POSTGRESQL_USER") != "" {
		options += "-U" + os.Getenv("POSTGRESQL_USER")
	} else {
		options += "-U postgres"
	}

	if os.Getenv("POSTGRESQL_HOST") != "" {
		options += " -h " + os.Getenv("POSTGRESQL_HOST")
	}

	if os.Getenv("POSTGRESQL_PORT") != "" {
		options += " -p " + os.Getenv("POSTGRESQL_PORT")
	} else {
		options += " -p 5432"
	}

	if os.Getenv("POSTGRESQL_DB") != "" {
		options += " " + os.Getenv("POSTGRESQL_DB")
	} else {
		options += " postgres"
	}

	if os.Getenv("BACKUP_DIR") != "" {
		options += " | " + "gzip > " + os.Getenv("BACKUP_DIR") + fileName
	} else {
		options += " | " + "gzip > /var/lib/postgresql/backups/" + fileName
	}

	return options
}

func postgresqlDump() {
	fileName := fileNameGenerate()
	options := generatePostgresqlDumpOptions(fileName)

	// pg_dump -U postgres -h 127.0.0.1 -p 5432 docker | gzip > /var/lib/postgresql/backups/filename.sql.gz
	cmd := exec.Command("/bin/sh",
		"-c",
		"pg_dump "+options)

	_, err := cmd.StdoutPipe()
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		sendToHorn("[PostgreSQL 📦] Возникли проблемы с бэкапом базы! ❌\nПодробнее можно узнать в Sentry! 🐞")
		log.Fatal(err)
	}

	var waitStatus syscall.WaitStatus
	if err := cmd.Run(); err != nil {
		if err != nil {
			raven.CaptureErrorAndWait(err, nil)
			sendToHorn("[PostgreSQL 📦] Возникли проблемы с бэкапом базы! ❌\nПодробнее можно узнать в Sentry! 🐞")
			log.Fatal(err)
		}
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus = exitError.Sys().(syscall.WaitStatus)
			fmt.Printf("Output: %s\n", []byte(fmt.Sprintf("%d", waitStatus.ExitStatus())))
		}
	} else {
		// Success
		waitStatus = cmd.ProcessState.Sys().(syscall.WaitStatus)
		fmt.Printf("Output: %s\n", []byte(fmt.Sprintf("%d", waitStatus.ExitStatus())))
	}

	file := os.Getenv("BACKUP_DIR") + fileName
	_, err = os.Stat(file)

	// See if the file exists.
	if os.IsNotExist(err) {
		raven.CaptureErrorAndWait(err, nil)
		sendToHorn("[PostgreSQL 📦] Возникли проблемы с бэкапом базы! ❌\nПодробнее можно узнать в Sentry! 🐞")
		log.Fatal(err)
	}

	sendToHorn("[PostgreSQL 📦] База была успешно забэкаплена! ✅")
}

func isOlder(t time.Time) bool {
	rotateParsedFromEnv, err := getenvInt32("ROTATED_TIME_IN_HOURS")
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}
	var rotateTimeInHours = time.Duration(rotateParsedFromEnv) * time.Hour
	return time.Now().Sub(t) > rotateTimeInHours
}

func findOlderFiles(dir string) (files []os.FileInfo, err error) {
	tmpfiles, err := ioutil.ReadDir(dir)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}

	for _, file := range tmpfiles {
		if file.Mode().IsRegular() {
			if isOlder(file.ModTime()) {
				files = append(files, file)
			}
		}
	}
	return
}

func gzTypeFileChecking(filename string) string {
	buf, _ := ioutil.ReadFile(os.Getenv("BACKUP_DIR") + filename)

	kind, _ := filetype.Match(buf)
	if kind == filetype.Unknown {
		fmt.Println("Unknown file type")
		return "unknown"
	}

	return kind.Extension
}

func deleteFile(fileName string) {
	var err = os.Remove(os.Getenv("BACKUP_DIR") + fileName)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}
}

func cleaner() {
	files, err := findOlderFiles(os.Getenv("BACKUP_DIR"))
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}
	for _, file := range files {
		fileType := gzTypeFileChecking(file.Name())
		if fileType == "gz" {
			deleteFile(file.Name())
			sendToHorn(fmt.Sprintf("[PostgreSQL 📦 - 👴🏿] Старый бэкап [%s] был успешно удален! ✅", file.Name()))
		}
	}
}

func makeBackup() {
	postgresqlDump()
	cleaner()
}

func initFoldersForBackups() {
	_, err := os.Stat(os.Getenv("BACKUP_DIR"))

	if os.IsNotExist(err) {
		errDir := os.MkdirAll(os.Getenv("BACKUP_DIR"), 0755)
		if errDir != nil {
			raven.CaptureErrorAndWait(err, nil)
			log.Fatal(err)
		}
	}
}

func tasks() {
	initFoldersForBackups()

	gocron.Every(1).Hour().From(gocron.NextTick()).Do(makeBackup)

	gocron.Every(1).Day().At("2:00").Do(makeBackup)

	// remove, clear and next_run
	_, time := gocron.NextRun()
	fmt.Println(time)

	// function Start start all the pending jobs
	<-gocron.Start()
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	appEnv := os.Getenv("APP_ENV")

	if appEnv == "production" {
		raven.SetDSN(os.Getenv("SENTRY_DSN"))
	}

	tasks()
}
