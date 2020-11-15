package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/getsentry/raven-go"
	"github.com/h2non/filetype"
	"github.com/ivahaev/russian-time"
	"github.com/joho/godotenv"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func getTemplate(fileName string, funcmap template.FuncMap, data interface{}) (result string, err error) {
	template, err := template.New(fileName).Funcs(funcmap).ParseFiles("templates/" + fileName)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Panic(err)
	}

	var tpl bytes.Buffer
	if err := template.Execute(&tpl, data); err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Panic(err)
		panic(err)
	}

	result = tpl.String()

	return
}

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

func dumpError() {
	type Info struct {
		Status string
	}

	templateData := Info{
		Status: "Dump error!",
	}

	funcmap := template.FuncMap{
		"weekDay":     weekDay,
		"hourWithMin": hourWithMin,
	}

	text, err := getTemplate("unsuccessful_backup.gohtml", funcmap, templateData)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Panic(err)
	}
	sendToHorn(text)
}

func dumpSuccess() {
	type Info struct {
		Status string
	}

	templateData := Info{
		Status: "Dump successful!",
	}

	funcmap := template.FuncMap{
		"weekDay":     weekDay,
		"hourWithMin": hourWithMin,
	}

	text, err := getTemplate("successful_backup.gohtml", funcmap, templateData)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Panic(err)
	}
	sendToHorn(text)
}

func cleanerSuccess(fileName string) {
	type Info struct {
		FileName string
	}

	templateData := Info{
		FileName: fileName,
	}

	funcmap := template.FuncMap{
		"weekDay":     weekDay,
		"hourWithMin": hourWithMin,
	}

	text, err := getTemplate("successful_cleaner.gohtml", funcmap, templateData)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Panic(err)
	}
	sendToHorn(text)
}

func fileNameGenerate() string {
	currentTime := time.Now()
	filename := os.Getenv("POSTGRESQL_DB") + "-" + currentTime.Format("2006_01_02_15_04_05") + ".sql.gz"
	return filename

}

func postgresqlDump() {
	fileName := fileNameGenerate()
	cmd := exec.Command("/bin/sh",
		"-c",
		"pg_dump "+"-U"+os.Getenv("POSTGRESQL_USER")+" -h "+os.Getenv("POSTGRESQL_HOST")+" -p "+os.Getenv("POSTGRESQL_PORT")+" "+os.Getenv("POSTGRESQL_DB") + " | "+"gzip > "+os.Getenv("BACKUP_DIR")+fileName)

	_, err := cmd.StdoutPipe()
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		dumpError()
		log.Fatal(err)
	}

	var waitStatus syscall.WaitStatus
	if err := cmd.Run(); err != nil {
		if err != nil {
			raven.CaptureErrorAndWait(err, nil)
			dumpError()
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
		dumpError()
		log.Fatal(err)
	}

	dumpSuccess()
}

func isOlder(t time.Time) bool {
	rotateParsedFromEnv, err := getenvInt32("ROTATED_TIME_IN_HOURS")
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		log.Fatal(err)
	}
	var rotateTimeInHours = time.Duration(rotateParsedFromEnv)*time.Hour
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
			cleanerSuccess(file.Name())
		}
	}
}

func makeBackup() {
	postgresqlDump()
	cleaner()
}

func initFoldersForBackups()  {
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
	makeBackup()
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
