package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func executeCommand(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCommand("test")
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetArgs(args)
	err := root.Execute()
	return output.String(), err
}

func TestInvoicesListUsesVoucherListDefaults(t *testing.T) {
	t.Setenv("LEXWARE_API_KEY", "secret")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/voucherlist" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("voucherType"); got != "invoice" {
			t.Fatalf("voucherType = %q", got)
		}
		if got := r.URL.Query().Get("voucherStatus"); got != "any" {
			t.Fatalf("voucherStatus = %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("Authorization = %q", got)
		}
		io.WriteString(w, `{"content":[],"first":true,"last":true,"totalPages":1,"totalElements":0,"numberOfElements":0,"size":25,"number":0}`)
	}))
	defer server.Close()

	output, err := executeCommand(t, "--base-url", server.URL, "--json", "invoices", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, `"content":[]`) {
		t.Fatalf("output = %q", output)
	}
}

func TestInvoiceCreateDryRunDoesNotNeedCredentials(t *testing.T) {
	t.Setenv("LEXWARE_API_KEY", "")
	path := filepath.Join(t.TempDir(), "invoice.json")
	if err := os.WriteFile(path, []byte(`{"voucherDate":"2026-08-31T00:00:00.000+02:00"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "--json", "invoices", "create", "--from", path, "--finalize", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("output is not JSON: %q (%v)", output, err)
	}
	if result["dryRun"] != true || result["finalize"] != true {
		t.Fatalf("result = %#v", result)
	}
}

func TestContactUpdateFetchesCurrentVersion(t *testing.T) {
	t.Setenv("LEXWARE_API_KEY", "secret")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			io.WriteString(w, `{"id":"contact-1","version":7}`)
		case http.MethodPut:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["version"] != float64(7) {
				t.Fatalf("version = %#v", body["version"])
			}
			io.WriteString(w, `{"id":"contact-1","version":8}`)
		default:
			t.Fatalf("method = %s", r.Method)
		}
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "contact.json")
	if err := os.WriteFile(path, []byte(`{"roles":{"customer":{}},"company":{"name":"Example"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "--base-url", server.URL, "--json", "contacts", "update", "contact-1", "--from", path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, `"version":8`) {
		t.Fatalf("output = %q", output)
	}
}

func TestVoucherUploadDryRunValidatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipt.pdf")
	if err := os.WriteFile(path, []byte("%PDF-test"), 0o600); err != nil {
		t.Fatal(err)
	}
	output, err := executeCommand(t, "--json", "vouchers", "upload", path, "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, `"type":"voucher"`) {
		t.Fatalf("output = %q", output)
	}
}
