package credentials

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	serviceName = "lexware-cli"
	accountName = "api-key"
	envName     = "LEXWARE_API_KEY"
)

var ErrNotConfigured = errors.New("kein API-Key konfiguriert; führe zuerst `num auth set` aus")

type Resolved struct {
	Token  string
	Source string
}

func Resolve() (Resolved, error) {
	if token := strings.TrimSpace(os.Getenv(envName)); token != "" {
		return Resolved{Token: token, Source: envName}, nil
	}

	token, err := keyring.Get(serviceName, accountName)
	if errors.Is(err, keyring.ErrNotFound) {
		return Resolved{}, ErrNotConfigured
	}
	if err != nil {
		return Resolved{}, fmt.Errorf("API-Key konnte nicht aus dem System-Schlüsselbund gelesen werden: %w", err)
	}
	if strings.TrimSpace(token) == "" {
		return Resolved{}, ErrNotConfigured
	}
	return Resolved{Token: token, Source: "System-Schlüsselbund"}, nil
}

func Set(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("API-Key darf nicht leer sein")
	}
	if err := keyring.Set(serviceName, accountName, token); err != nil {
		return fmt.Errorf("API-Key konnte nicht im System-Schlüsselbund gespeichert werden: %w", err)
	}
	return nil
}

func Delete() error {
	err := keyring.Delete(serviceName, accountName)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("API-Key konnte nicht aus dem System-Schlüsselbund entfernt werden: %w", err)
	}
	return nil
}
