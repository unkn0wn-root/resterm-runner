package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/unkn0wn-root/resterm/headless"
)

type Opt struct {
	Use     string
	Version string
	Commit  string
	Date    string
	Stdout  io.Writer
	Stderr  io.Writer
}

type cmd struct {
	fs          *flag.FlagSet
	use         string
	version     string
	commit      string
	date        string
	stdout      io.Writer
	stderr      io.Writer
	filePath    string
	envName     string
	envFile     string
	artifactDir string
	reportJSON  string
	reportJUnit string
	workspace   string
	stateDir    string
	timeout     time.Duration
	insecure    bool
	follow      bool
	proxyURL    string
	recursive   bool
	persistAuth bool
	persistVars bool
	history     bool
	reqName     string
	workflow    string
	tag         string
	compareRaw  string
	compareBase string
	all         bool
	profile     bool
	showVersion bool
}

func Run(args []string, opt Opt) error {
	if err := validateWriters(opt); err != nil {
		return err
	}
	c := newCmd(opt)
	if err := c.fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return ExitErr{Err: err, Code: 2}
	}
	return c.run()
}

func newCmd(opt Opt) *cmd {
	c := &cmd{
		use:     strings.TrimSpace(opt.Use),
		version: strings.TrimSpace(opt.Version),
		commit:  strings.TrimSpace(opt.Commit),
		date:    strings.TrimSpace(opt.Date),
		stdout:  opt.Stdout,
		stderr:  opt.Stderr,
	}
	if c.use == "" {
		c.use = "resterm-runner"
	}

	fs := flag.NewFlagSet(c.use, flag.ContinueOnError)
	fs.SetOutput(c.stderr)
	c.fs = fs
	c.bind()
	fs.Usage = c.usage
	return c
}

func validateWriters(opt Opt) error {
	if opt.Stdout == nil || opt.Stderr == nil {
		return headless.ErrNilWriter
	}
	return nil
}

func (c *cmd) bind() {
	c.fs.StringVar(&c.filePath, "file", "", "Path to .http/.rest file to run")
	c.fs.StringVar(&c.envName, "env", "", "Environment name to use")
	c.fs.StringVar(&c.envFile, "env-file", "", "Path to environment file")
	c.fs.StringVar(&c.artifactDir, "artifact-dir", "", "Write run artifacts to the given directory")
	c.fs.StringVar(&c.reportJSON, "report-json", "", "Write JSON report to the given path")
	c.fs.StringVar(&c.reportJUnit, "report-junit", "", "Write JUnit XML report to the given path")
	c.fs.StringVar(&c.workspace, "workspace", "", "Workspace directory to scan for request files")
	c.fs.StringVar(&c.stateDir, "state-dir", "", "Directory for persisted runner state")
	c.fs.DurationVar(&c.timeout, "timeout", 30*time.Second, "Request timeout")
	c.fs.BoolVar(&c.insecure, "insecure", false, "Skip TLS certificate verification")
	c.fs.BoolVar(&c.follow, "follow", true, "Follow redirects")
	c.fs.StringVar(&c.proxyURL, "proxy", "", "HTTP proxy URL")
	c.fs.BoolVar(&c.recursive, "recursive", false, "Recursively scan workspace for request files")
	c.fs.BoolVar(&c.persistVars, "persist-globals", false, "Persist runtime globals and file vars between runs")
	c.fs.BoolVar(&c.persistAuth, "persist-auth", false, "Persist auth caches between runs")
	c.fs.BoolVar(&c.history, "history", false, "Persist request history for runner executions")
	c.fs.StringVar(&c.reqName, "request", "", "Request name to run")
	c.fs.StringVar(&c.workflow, "workflow", "", "Workflow name to run")
	c.fs.StringVar(&c.tag, "tag", "", "Run requests with the given tag")
	c.fs.StringVar(&c.compareRaw, "compare", "", "Compare environments (comma/space separated)")
	c.fs.StringVar(&c.compareBase, "compare-base", "", "Baseline environment for --compare")
	c.fs.BoolVar(&c.all, "all", false, "Run all requests in the file")
	c.fs.BoolVar(&c.profile, "profile", false, "Profile the selected request run(s)")
	c.fs.BoolVar(&c.showVersion, "version", false, "Show resterm-runner version")
}

func (c *cmd) usage() {
	fmt.Fprintf(c.stderr, "Usage: %s [flags] [file]\n", c.use)
	fmt.Fprintln(c.stderr, "")
	fmt.Fprintln(c.stderr, "Flags:")
	c.fs.PrintDefaults()
}

func (c *cmd) run() error {
	if c.showVersion {
		return c.printVersion()
	}
	if c.filePath == "" && len(c.fs.Args()) > 0 {
		c.filePath = c.fs.Arg(0)
	}
	if len(c.fs.Args()) > 1 {
		return ExitErr{
			Err:  fmt.Errorf("run: unexpected args: %s", strings.Join(c.fs.Args()[1:], " ")),
			Code: 2,
		}
	}
	if strings.TrimSpace(c.filePath) == "" {
		return ExitErr{Err: errors.New("run: --file is required"), Code: 2}
	}

	targets, err := parseCompare(c.compareRaw)
	if err != nil {
		return ExitErr{
			Err:  fmt.Errorf("run: invalid --compare value: %w", err),
			Code: 2,
		}
	}

	rep, err := headless.Run(context.Background(), headless.Opt{
		Version:        c.version,
		FilePath:       c.filePath,
		Workspace:      c.workspace,
		Recursive:      c.recursive,
		ArtifactDir:    c.artifactDir,
		StateDir:       c.stateDir,
		PersistGlobals: c.persistVars,
		PersistAuth:    c.persistAuth,
		History:        c.history,
		EnvName:        c.envName,
		EnvFile:        c.envFile,
		CompareTargets: targets,
		CompareBase:    c.compareBase,
		Profile:        c.profile,
		HTTP: headless.HTTPOpt{
			Timeout:  c.timeout,
			Follow:   boolPtr(c.follow),
			Insecure: c.insecure,
			Proxy:    c.proxyURL,
		},
		Select: headless.Select{
			Request:  c.reqName,
			Workflow: c.workflow,
			Tag:      c.tag,
			All:      c.all,
		},
	})
	if err != nil {
		if headless.IsUsageError(err) {
			return ExitErr{Err: fmt.Errorf("run: %w", err), Code: 2}
		}
		return err
	}
	if err := rep.WriteText(c.stdout); err != nil {
		return fmt.Errorf("run: write output: %w", err)
	}
	if err := writeJSON(c.reportJSON, rep); err != nil {
		return fmt.Errorf("run: write json report: %w", err)
	}
	if err := writeJUnit(c.reportJUnit, rep); err != nil {
		return fmt.Errorf("run: write junit report: %w", err)
	}
	if rep.Failed > 0 {
		return ExitErr{Err: errors.New("one or more requests failed"), Code: 1}
	}
	return nil
}

func writeJSON(path string, rep *headless.Report) error {
	return writeReport(path, func(w io.Writer) error {
		return rep.WriteJSON(w)
	})
}

func writeJUnit(path string, rep *headless.Report) error {
	return writeReport(path, func(w io.Writer) error {
		return rep.WriteJUnit(w)
	})
}

func writeReport(path string, fn func(io.Writer) error) (err error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	return fn(f)
}

func boolPtr(v bool) *bool {
	return &v
}

func (c *cmd) printVersion() error {
	if _, err := fmt.Fprintf(c.stdout, "%s %s\n", c.use, c.version); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.stdout, "  commit: %s\n", c.commit); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.stdout, "  built:  %s\n", c.date); err != nil {
		return err
	}
	sum, err := executableChecksum()
	if err != nil {
		_, werr := fmt.Fprintf(c.stdout, "  sha256: unavailable (%v)\n", err)
		return werr
	}
	_, err = fmt.Fprintf(c.stdout, "  sha256: %s\n", sum)
	return err
}

func executableChecksum() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	f, err := os.Open(exe)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
