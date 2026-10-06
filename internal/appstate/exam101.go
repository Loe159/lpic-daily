package appstate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func Exam101AssessmentPath() (string, error) {
	base, err := stateBase()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(base, "exam101-assessment"), nil
}

func Exam101AssessmentPassed() (bool, error) {
	path, err := Exam101AssessmentPath()
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	switch strings.TrimSpace(string(data)) {
	case "pass":
		return true, nil
	case "fail", "":
		return false, nil
	default:
		return false, errors.New("invalid Exam 101 assessment state")
	}
}

func SaveExam101AssessmentPassed(passed bool) error {
	path, err := Exam101AssessmentPath()
	if err != nil {
		return err
	}
	value := "fail\n"
	if passed {
		value = "pass\n"
	}
	return os.WriteFile(path, []byte(value), 0o600)
}
