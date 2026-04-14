package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unkn0wn-root/resterm/headless"
)

func TestRunHelpFlag(t *testing.T) {
	var out strings.Builder
	var errOut strings.Builder

	err := Run([]string{"--help"}, Opt{
		Use:    "resterm-runner",
		Stdout: &out,
		Stderr: &errOut,
	})
	if err != nil {
		t.Fatalf("Run --help: %v", err)
	}
	if !isBlank(out.String()) {
		t.Fatalf("expected empty stdout, got %q", out.String())
	}
	if !strings.Contains(errOut.String(), "Usage: resterm-runner [flags] [file]") {
		t.Fatalf("expected usage in stderr, got %q", errOut.String())
	}
}

func TestRunVersionFlag(t *testing.T) {
	var out strings.Builder
	var errOut strings.Builder

	err := Run([]string{"--version"}, Opt{
		Use:     "resterm-runner",
		Version: "v1.2.3",
		Commit:  "abc1234",
		Date:    "2026-04-13 11:22:33 UTC",
		Stdout:  &out,
		Stderr:  &errOut,
	})
	if err != nil {
		t.Fatalf("Run --version: %v", err)
	}
	if !isBlank(errOut.String()) {
		t.Fatalf("expected empty stderr, got %q", errOut.String())
	}
	got := out.String()
	if !strings.Contains(got, "resterm-runner v1.2.3") {
		t.Fatalf("expected version header, got %q", got)
	}
	if !strings.Contains(got, "commit: abc1234") {
		t.Fatalf("expected commit line, got %q", got)
	}
	if !strings.Contains(got, "built:  2026-04-13 11:22:33 UTC") {
		t.Fatalf("expected build line, got %q", got)
	}
	if !strings.Contains(got, "sha256: ") {
		t.Fatalf("expected sha256 line, got %q", got)
	}
}

func TestRunRequestSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	dir := t.TempDir()
	file := filepath.Join(dir, "ok.http")
	src := fmt.Sprintf("# @name ok\nGET %s\n", srv.URL)
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var out strings.Builder
	var errOut strings.Builder
	err := Run([]string{"--file", file}, Opt{
		Use:    "resterm-runner",
		Stdout: &out,
		Stderr: &errOut,
	})
	if err != nil {
		t.Fatalf("Run request: %v", err)
	}
	if !isBlank(errOut.String()) {
		t.Fatalf("expected empty stderr, got %q", errOut.String())
	}
	if !strings.Contains(out.String(), "PASS GET ok") {
		t.Fatalf("expected pass output, got %q", out.String())
	}
}

func TestRunFailureExitCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	dir := t.TempDir()
	file := filepath.Join(dir, "bad.http")
	src := strings.Join([]string{
		"# @name bad",
		"# @assert response.statusCode == 201",
		fmt.Sprintf("GET %s", srv.URL),
		"",
	}, "\n")
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var out strings.Builder
	err := Run([]string{"--file", file}, Opt{
		Use:    "resterm-runner",
		Stdout: &out,
		Stderr: &strings.Builder{},
	})
	if err == nil {
		t.Fatal("expected run failure")
	}
	if code := ExitCode(err); code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(out.String(), "FAIL GET bad") {
		t.Fatalf("expected fail output, got %q", out.String())
	}
}

func TestRunSelectorErrorReturnsCodeTwo(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "many.http")
	src := strings.Join([]string{
		"### One",
		"GET https://example.com/one",
		"",
		"### Two",
		"GET https://example.com/two",
		"",
	}, "\n")
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	err := Run([]string{"--file", file}, Opt{
		Use:    "resterm-runner",
		Stdout: io.Discard,
		Stderr: io.Discard,
	})
	if err == nil {
		t.Fatal("expected selector error")
	}
	if code := ExitCode(err); code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}

func TestRunNilWriterReturnsErrNilWriter(t *testing.T) {
	err := Run(nil, Opt{
		Use:    "resterm-runner",
		Stdout: io.Discard,
	})
	if !errors.Is(err, headless.ErrNilWriter) {
		t.Fatalf("Run with nil stderr: got %v want %v", err, headless.ErrNilWriter)
	}

	err = Run(nil, Opt{
		Use:    "resterm-runner",
		Stderr: io.Discard,
	})
	if !errors.Is(err, headless.ErrNilWriter) {
		t.Fatalf("Run with nil stdout: got %v want %v", err, headless.ErrNilWriter)
	}
}

func TestRunHeadlessUsageErrorReturnsCodeTwo(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "env.http")
	src := strings.Join([]string{
		"# @name ok",
		"GET https://example.com",
		"",
	}, "\n")
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	err := Run([]string{
		"--file", file,
		"--env", "$shared",
	}, Opt{
		Use:    "resterm-runner",
		Stdout: io.Discard,
		Stderr: io.Discard,
	})
	if err == nil {
		t.Fatal("expected usage error")
	}
	if code := ExitCode(err); code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !headless.IsUsageError(err) {
		t.Fatalf("expected headless usage error, got %v", err)
	}
}

func TestRunCompareWritesJSONReport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dev":
			fmt.Fprint(w, `{"env":"dev"}`)
		case "/stage":
			fmt.Fprint(w, `{"env":"stage"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dir := t.TempDir()
	file := filepath.Join(dir, "compare.http")
	envFile := filepath.Join(dir, "rest-client.env.json")
	report := filepath.Join(dir, "out", "report.json")
	src := strings.Join([]string{
		"# @name cmp",
		fmt.Sprintf("GET %s/{{path}}", srv.URL),
		"",
	}, "\n")
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	envs := `{
  "dev": {"path": "dev"},
  "stage": {"path": "stage"}
}`
	if err := os.WriteFile(envFile, []byte(envs), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	var out strings.Builder
	var errOut strings.Builder
	err := Run([]string{
		"--file", file,
		"--env-file", envFile,
		"--compare", "dev,stage",
		"--compare-base", "stage",
		"--report-json", report,
	}, Opt{
		Use:    "resterm-runner",
		Stdout: &out,
		Stderr: &errOut,
	})
	if err != nil {
		t.Fatalf("Run compare: %v", err)
	}
	if !isBlank(errOut.String()) {
		t.Fatalf("expected empty stderr, got %q", errOut.String())
	}
	if !strings.Contains(out.String(), "PASS COMPARE cmp") {
		t.Fatalf("expected compare output, got %q", out.String())
	}

	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var got struct {
		Results []struct {
			Kind    string `json:"kind"`
			Name    string `json:"name"`
			Status  string `json:"status"`
			Compare struct {
				Baseline string `json:"baseline"`
			} `json:"compare"`
			Steps []struct {
				Environment string `json:"environment"`
				Status      string `json:"status"`
			} `json:"steps"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if len(got.Results) != 1 {
		t.Fatalf("expected one compare result, got %+v", got.Results)
	}
	item := got.Results[0]
	if item.Kind != "compare" || item.Name != "cmp" || item.Status != "pass" {
		t.Fatalf("unexpected compare result: %+v", item)
	}
	if item.Compare.Baseline != "stage" {
		t.Fatalf("unexpected compare baseline: %+v", item.Compare)
	}
	if len(item.Steps) != 2 || item.Steps[0].Environment != "dev" || item.Steps[1].Environment != "stage" {
		t.Fatalf("unexpected compare steps: %+v", item.Steps)
	}
}
