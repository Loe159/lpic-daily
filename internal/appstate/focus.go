package appstate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	StudyFocusExam101 = "101"
	StudyFocusAll     = "all"
)

func StudyFocusPath() (string, error) {
	base, err := stateBase()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(base, "study-focus"), nil
}

func LoadStudyFocus() (string, error) {
	path, err := StudyFocusPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return StudyFocusExam101, nil
	}
	if err != nil {
		return "", err
	}
	focus := strings.TrimSpace(string(data))
	if focus != StudyFocusExam101 && focus != StudyFocusAll {
		return "", errors.New("invalid study focus")
	}
	return focus, nil
}

func SaveStudyFocus(focus string) error {
	if focus != StudyFocusExam101 && focus != StudyFocusAll {
		return errors.New("study focus must be 101 or all")
	}
	path, err := StudyFocusPath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(focus+"\n"), 0o600)
}
