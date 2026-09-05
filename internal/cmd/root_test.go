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
	"sync/atomic"
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

func TestPostingCategoriesListUsesOfficialEndpoint(t *testing.T) {
	t.Setenv("LEXWARE_API_KEY", "secret")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s", r.Method)
		}
		if r.URL.Path != "/v1/posting-categories" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Fatalf("Authorization = %q", got)
		}
		io.WriteString(w, `[{"id":"category-1","name":"Fremdleistungen §13b","type":"outgo"}]`)
	}))
	defer server.Close()

	output, err := executeCommand(t, "--base-url", server.URL, "--json", "posting-categories", "list")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, `"name":"Fremdleistungen §13b"`) {
		t.Fatalf("output = %q", output)
	}
}

func TestVouchersListYearFetchesEveryPage(t *testing.T) {
	t.Setenv("LEXWARE_API_KEY", "secret")
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("voucherDateFrom"); got != "2025-01-01" {
			t.Fatalf("voucherDateFrom = %q", got)
		}
		if got := r.URL.Query().Get("voucherDateTo"); got != "2025-12-31" {
			t.Fatalf("voucherDateTo = %q", got)
		}
		if got := r.URL.Query().Get("size"); got != "250" {
			t.Fatalf("size = %q", got)
		}
		page := r.URL.Query().Get("page")
		switch calls.Add(1) {
		case 1:
			if page != "0" {
				t.Fatalf("first page = %q", page)
			}
			io.WriteString(w, `{"content":[{"id":"one"}],"first":true,"last":false,"totalPages":2,"totalElements":2,"numberOfElements":1,"size":250,"number":0}`)
		case 2:
			if page != "1" {
				t.Fatalf("second page = %q", page)
			}
			io.WriteString(w, `{"content":[{"id":"two"}],"first":false,"last":true,"totalPages":2,"totalElements":2,"numberOfElements":1,"size":250,"number":1}`)
		default:
			t.Fatalf("unexpected request %d", calls.Load())
		}
	}))
	defer server.Close()

	output, err := executeCommand(t, "--base-url", server.URL, "--json", "vouchers", "list", "--year", "2025")
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Content []map[string]any `json:"content"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 || len(result.Content) != 2 {
		t.Fatalf("calls = %d, result = %#v", calls.Load(), result)
	}
}

func TestVouchersListYearRejectsExplicitDateRange(t *testing.T) {
	output, err := executeCommand(t, "vouchers", "list", "--year", "2025", "--voucher-date-from", "2025-02-01")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "kann nicht") {
		t.Fatalf("error = %v, output = %q", err, output)
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

func TestVoucherDownloadUsesFileEndpointAndWritesSecurely(t *testing.T) {
	t.Setenv("LEXWARE_API_KEY", "secret")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/files/file-1" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "*/*" {
			t.Fatalf("Accept = %q", got)
		}
		w.Header().Set("Content-Disposition", `attachment; filename="twilio-receipt.pdf"`)
		w.Header().Set("Content-Type", "application/pdf")
		io.WriteString(w, "%PDF-test")
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "download.pdf")
	output, err := executeCommand(t, "--base-url", server.URL, "--json", "vouchers", "download", "file-1", "--output", target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, `"ok":true`) || !strings.Contains(output, target) {
		t.Fatalf("output = %q", output)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "%PDF-test" {
		t.Fatalf("data = %q", data)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o", got)
	}
}

func TestVoucherDownloadDoesNotOverwriteWithoutForce(t *testing.T) {
	t.Setenv("LEXWARE_API_KEY", "secret")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "replacement")
	}))
	defer server.Close()

	target := filepath.Join(t.TempDir(), "existing.pdf")
	if err := os.WriteFile(target, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := executeCommand(t, "--base-url", server.URL, "vouchers", "download", "file-1", "--output", target)
	if err == nil || !strings.Contains(err.Error(), "existiert bereits") {
		t.Fatalf("error = %v", err)
	}
	data, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(data) != "original" {
		t.Fatalf("data = %q", data)
	}
}

func TestNumEntrypointAndCompletions(t *testing.T) {
	isolatedAgentHome(t)
	version, err := executeCommand(t, "--version")
	if err != nil || version != "num test\n" {
		t.Fatalf("version = %q, err = %v", version, err)
	}
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			completion, err := executeCommand(t, "completion", shell)
			if err != nil || !strings.Contains(completion, "num") || strings.Contains(completion, "lexware") {
				t.Fatalf("completion is not for num: %q, err = %v", completion, err)
			}
		})
	}
}
