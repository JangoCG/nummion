package cmd

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func writeDownload(resp *http.Response, requestedPath, fallbackName string, force bool) (string, error) {
	defer resp.Body.Close()
	target := requestedPath
	if target == "" {
		target = suggestedFilename(resp.Header.Get("Content-Disposition"), fallbackName)
	}
	target = filepath.Clean(target)
	flags := os.O_WRONLY | os.O_CREATE
	if force {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}
	file, err := os.OpenFile(target, flags, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("Datei %q existiert bereits; verwende --force zum Überschreiben", target)
		}
		return "", fmt.Errorf("Zieldatei konnte nicht geöffnet werden: %w", err)
	}
	complete := false
	defer func() {
		file.Close()
		if !complete {
			_ = os.Remove(target)
		}
	}()
	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", fmt.Errorf("Download konnte nicht gespeichert werden: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("Download konnte nicht abgeschlossen werden: %w", err)
	}
	complete = true
	return target, nil
}

func suggestedFilename(contentDisposition, fallback string) string {
	if _, params, err := mime.ParseMediaType(contentDisposition); err == nil {
		if name := safeBaseName(params["filename"]); name != "" {
			return name
		}
	}
	if name := safeBaseName(fallback); name != "" {
		return name
	}
	return "lexware-download"
}

func safeBaseName(value string) string {
	value = strings.ReplaceAll(value, "\x00", "")
	name := filepath.Base(strings.TrimSpace(value))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return ""
	}
	return name
}
