package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/getsentry/raven-go"
	"github.com/h2non/filetype"
	"github.com/joho/godotenv"
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
		log.Println(err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Println(err)

		fmt.Println(resp)
	}
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
		options += " | " + "gzip > " + os.Getenv("BACKUP_DIR") + "daily/" + fileName
	} else {
		options += " | " + "gzip > /var/lib/postgresql/backups/daily/" + fileName
	}

	return options
}

func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func Round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

func HumanFileSize(size float64) string {
	var (
		suffixes [5]string
	)

	fmt.Println(size)
	suffixes[0] = "B"
	suffixes[1] = "KB"
	suffixes[2] = "MB"
	suffixes[3] = "GB"
	suffixes[4] = "TB"

	base := math.Log(size) / math.Log(1024)
	getSize := Round(math.Pow(1024, base-math.Floor(base)), .5, 2)
	fmt.Println(int(math.Floor(base)))
	getSuffix := suffixes[int(math.Floor(base))]
	return strconv.FormatFloat(getSize, 'f', -1, 64) + " " + string(getSuffix)
}

func postgresqlDump() {
	fileName := fileNameGenerate()
	dbPassword := os.Getenv("POSTGRESQL_PASSWORD")
	options := generatePostgresqlDumpOptions(fileName)

	// pg_dump -U postgres -h 127.0.0.1 -p 5432 docker | gzip > /var/lib/postgresql/backups/filename.sql.gz
	cmd := exec.Command("/bin/sh", "-c", "pg_dump "+options)
	additionalEnv := "PGPASSWORD=" + dbPassword
	newEnv := append(os.Environ(), additionalEnv)
	cmd.Env = newEnv
	_, err := cmd.StdoutPipe()
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		sendToHorn("[PostgreSQL 📦] Возникли проблемы с бэкапом базы! ❌\nПодробнее можно узнать в Sentry! 🐞")
		log.Fatal(err)
	}

	var waitStatus syscall.WaitStatus
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus = exitError.Sys().(syscall.WaitStatus)
			fmt.Printf("Output: %s\n", []byte(fmt.Sprintf("%d", waitStatus.ExitStatus())))
		}

		raven.CaptureErrorAndWait(err, nil)
		sendToHorn("[PostgreSQL 📦] Возникли проблемы с бэкапом базы! ❌\nПодробнее можно узнать в Sentry! 🐞")
		log.Fatal(err)
	} else {
		// Success
		waitStatus = cmd.ProcessState.Sys().(syscall.WaitStatus)
		fmt.Printf("Output: %s\n", []byte(fmt.Sprintf("%d", waitStatus.ExitStatus())))
	}

	file := os.Getenv("BACKUP_DIR") + "daily/" + fileName
	dailyBackupFile, err := os.Stat(file)
	dailyBackupFileSize := dailyBackupFile.Name() + " - " + HumanFileSize(float64(dailyBackupFile.Size()))

	// See if the file exists.
	if os.IsNotExist(err) {
		raven.CaptureErrorAndWait(err, nil)
		sendToHorn("[PostgreSQL 📦] Возникли проблемы с бэкапом базы! ❌\nПодробнее можно узнать в Sentry! 🐞")
		log.Fatal(err)
	}

	fmt.Println("[PostgreSQL 📦][DAILY ROTATOR " + dailyBackupFileSize + "] База была успешно забэкаплена! ✅")
	sendToHorn("[PostgreSQL 📦][DAILY ROTATOR " + dailyBackupFileSize + "] База была успешно забэкаплена! ✅")
	weeklyRotator(fileName)
	monthlyRotator(fileName)
}

func weeklyRotator(fileName string) {
	weekDay := int(time.Now().Weekday())
	if weekDay == 1 && os.Getenv("ROTATED_TIME_WEEKLY") != "" {
		_, err := copyFile(os.Getenv("BACKUP_DIR")+"daily/"+fileName, os.Getenv("BACKUP_DIR")+"weekly/"+"weekly_"+fileName)
		if err != nil {
			raven.CaptureErrorAndWait(err, nil)
			sendToHorn("[PostgreSQL 📦][WEEKLY ROTATOR] Возникли проблемы с бэкапом базы! ❌\nПодробнее можно узнать в Sentry! 🐞")
			log.Fatal(err)
		}
		sendToHorn("[PostgreSQL 📦][WEEKLY ROTATOR] База была успешно забэкаплена! ✅")
	}
}

func monthlyRotator(fileName string) {
	monthDay := time.Now().Day()
	if monthDay == 1 && os.Getenv("ROTATED_TIME_MONTHLY") != "" {
		_, err := copyFile(os.Getenv("BACKUP_DIR")+"daily/"+fileName, os.Getenv("BACKUP_DIR")+"monthly/"+"monthly_"+fileName)
		if err != nil {
			raven.CaptureErrorAndWait(err, nil)
			sendToHorn("[PostgreSQL 📦][MONTHLY ROTATOR] Возникли проблемы с бэкапом базы! ❌\nПодробнее можно узнать в Sentry! 🐞")
			log.Fatal(err)
		}
		sendToHorn("[PostgreSQL 📦][MONTHLY ROTATOR] База была успешно забэкаплена! ✅")
	}
}

func isOlderDaily(t time.Time) bool {
	rotateParsedFromEnv, err := getenvInt32("ROTATED_TIME_DAILY")
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}
	var rotateTimeInHours = time.Duration(rotateParsedFromEnv*24+24) * time.Hour
	return time.Now().Sub(t) > rotateTimeInHours
}

func findOlderFilesDaily(dir string) (files []os.FileInfo, err error) {
	tmpfiles, err := ioutil.ReadDir(dir)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}

	for _, file := range tmpfiles {
		if file.Mode().IsRegular() {
			if isOlderDaily(file.ModTime()) {
				files = append(files, file)
			}
		}
	}
	return
}

func isOlderWeekly(t time.Time) bool {
	rotateParsedFromEnv, err := getenvInt32("ROTATED_TIME_WEEKLY")
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}
	var rotateTimeInHours = time.Duration(rotateParsedFromEnv*168+24) * time.Hour
	return time.Now().Sub(t) > rotateTimeInHours
}

func findOlderFilesWeekly(dir string) (files []os.FileInfo, err error) {
	tmpfiles, err := ioutil.ReadDir(dir)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}

	for _, file := range tmpfiles {
		if file.Mode().IsRegular() {
			if isOlderWeekly(file.ModTime()) {
				files = append(files, file)
			}
		}
	}
	return
}

func findOlderFilesMonthly(dir string) (files []os.FileInfo, err error) {
	tmpfiles, err := ioutil.ReadDir(dir)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}

	for _, file := range tmpfiles {
		if file.Mode().IsRegular() {
			if isOlderMonthly(file.ModTime()) {
				files = append(files, file)
			}
		}
	}
	return
}

func isOlderMonthly(t time.Time) bool {
	rotateParsedFromEnv, err := getenvInt("ROTATED_TIME_MONTHLY")
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}
	currentTime := time.Now()
	var rotateTimeInHours = currentTime.AddDate(0, rotateParsedFromEnv, 0).Sub(currentTime)
	return time.Now().Sub(t) > rotateTimeInHours
}

func gzTypeFileChecking(filePath string) string {
	buf, _ := ioutil.ReadFile(filePath)

	kind, _ := filetype.Match(buf)
	if kind == filetype.Unknown {
		fmt.Println("Unknown file type")
		return "unknown"
	}

	return kind.Extension
}

func deleteFile(filePath string) {
	var err = os.Remove(filePath)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}
}

func cleanerDaily() {
	files, err := findOlderFilesDaily(os.Getenv("BACKUP_DIR") + "daily")
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}
	for _, file := range files {
		fileType := gzTypeFileChecking(os.Getenv("BACKUP_DIR") + "daily/" + file.Name())
		if fileType == "gz" {
			deleteFile(os.Getenv("BACKUP_DIR") + "daily/" + file.Name())
			sendToHorn(fmt.Sprintf("[PostgreSQL 📦 - 👴🏿] Старый бэкап [%s] был успешно удален! ✅", file.Name()))
		}
	}
}

func cleanerWeekly() {
	files, err := findOlderFilesWeekly(os.Getenv("BACKUP_DIR") + "weekly")
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}
	for _, file := range files {
		fileType := gzTypeFileChecking(os.Getenv("BACKUP_DIR") + "weekly/" + file.Name())
		if fileType == "gz" {
			deleteFile(os.Getenv("BACKUP_DIR") + "weekly/" + file.Name())
			sendToHorn(fmt.Sprintf("[PostgreSQL 📦 - 👴🏿] Старый бэкап [%s] был успешно удален! ✅", file.Name()))
		}
	}
}

func cleanerMonthly() {
	files, err := findOlderFilesMonthly(os.Getenv("BACKUP_DIR") + "monthly")
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}
	for _, file := range files {
		fileType := gzTypeFileChecking(os.Getenv("BACKUP_DIR") + "monthly/" + file.Name())
		if fileType == "gz" {
			deleteFile(os.Getenv("BACKUP_DIR") + "monthly/" + file.Name())
			sendToHorn(fmt.Sprintf("[PostgreSQL 📦 - 👴🏿] Старый бэкап [%s] был успешно удален! ✅", file.Name()))
		}
	}
}

func makeBackup() {
	postgresqlDump()
	cleanerDaily()
	cleanerWeekly()
	cleanerMonthly()
}

func initFoldersForBackups() {
	mainDir, err := os.Stat(os.Getenv("BACKUP_DIR"))
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(os.Getenv("BACKUP_DIR"), 0755)
		if errDir != nil {
			raven.CaptureErrorAndWait(err, nil)
			log.Println(mainDir)
			log.Fatal(err)
		}
	}

	daily, err := os.Stat(os.Getenv("BACKUP_DIR"))
	errDirDaily := os.MkdirAll(os.Getenv("BACKUP_DIR")+"daily", 0755)
	if errDirDaily != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Println(daily)
		log.Fatal(err)
	}

	weekly, err := os.Stat(os.Getenv("BACKUP_DIR"))
	errDirWeekly := os.MkdirAll(os.Getenv("BACKUP_DIR")+"weekly", 0755)
	if errDirWeekly != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Println(weekly)
		log.Fatal(err)
	}

	monthly, err := os.Stat(os.Getenv("BACKUP_DIR"))
	errDirMonthly := os.MkdirAll(os.Getenv("BACKUP_DIR")+"monthly", 0755)
	if errDirMonthly != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Println(monthly)
		log.Fatal(err)
	}
}

func tasks() {
	initFoldersForBackups()
	makeBackup()
}

func main() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	environmentPath := filepath.Join(dir, ".env")
	linuxEnvironmentPath := filepath.Join("/usr/local/etc/postgresql_backup", ".env")
	err = godotenv.Load(environmentPath)
	errLinuxConfigLoading := godotenv.Load(linuxEnvironmentPath)
	if err != nil && errLinuxConfigLoading != nil {
		log.Fatal("Error loading .env file \n Check .env in current directory or in /usr/local/etc/postgresql_backup")
	}

	appEnv := os.Getenv("APP_ENV")

	if appEnv == "production" {
		err := raven.SetDSN(os.Getenv("SENTRY_DSN"))
		if err != nil {
			log.Println(err)
		}
	}

	tasks()
}
