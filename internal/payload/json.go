package payload

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

func ReadJSON(path string, stdin io.Reader) ([]byte, map[string]any, error) {
	var (
		data []byte
		err  error
	)
	switch path {
	case "":
		return nil, nil, errors.New("--from ist erforderlich; verwende einen Dateipfad oder '-' für stdin")
	case "-":
		data, err = io.ReadAll(io.LimitReader(stdin, 10<<20))
	default:
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("JSON-Eingabe konnte nicht gelesen werden: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var object map[string]any
	if err := decoder.Decode(&object); err != nil {
		return nil, nil, fmt.Errorf("ungültiges JSON: %w", err)
	}
	if object == nil {
		return nil, nil, errors.New("JSON-Eingabe muss ein Objekt sein")
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return nil, nil, errors.New("JSON-Eingabe darf nur ein Objekt enthalten")
	}
	normalized, err := json.Marshal(object)
	if err != nil {
		return nil, nil, err
	}
	return normalized, object, nil
}

func MarshalObject(object map[string]any) ([]byte, error) {
	return json.Marshal(object)
}
