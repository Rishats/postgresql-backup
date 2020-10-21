package main

import (
	"errors"
	"github.com/getsentry/raven-go"
	"os"
	"strconv"
)

var ErrEnvVarEmpty = errors.New("getenv: environment variable empty")

func getenvStr(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		raven.CaptureErrorAndWait(ErrEnvVarEmpty, nil)
		return v, ErrEnvVarEmpty
	}
	return v, nil
}

func getenvInt(key string) (int, error) {
	s, err := getenvStr(key)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		return 0, err
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		return 0, err
	}
	return v, nil
}

func getenvInt32(key string) (int32, error) {
	s, err := getenvStr(key)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		return 0, err
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		return 0, err
	}
	return int32(v), nil
}

func getenvBool(key string) (bool, error) {
	s, err := getenvStr(key)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		return false, err
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		raven.CaptureErrorAndWait(err, nil)
		return false, err
	}
	return v, nil
}

