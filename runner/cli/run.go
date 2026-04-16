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
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/unkn0wn-root/resterm/headless"
)

var readBuildInfo = debug.ReadBuildInfo

type Opt struct {
	Use     string
	Version string
	Commit  string
	Date    string
	Stdout  io.Writer
	Stderr  io.Writer
	Context context.Context
}

type cmd struct {
	fs          *flag.FlagSet
	use         string
	version     string
	commit      string
	date        string
	stdout      io.Writer
	stderr      io.Writer
	ctx         context.Context
	filePath    string
	envName     string
	envFile     string
	artifactDir string
	reportJSON  string
	reportJUnit string
	workspace   string
	stateDir    string
	timeout     time.Duration
	runTimeout  time.Duration
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
	opt = opt.trimmed()
	opt = resolveBuildMeta(opt)
	c := &cmd{
		use:     opt.Use,
		version: opt.Version,
		commit:  opt.Commit,
		date:    opt.Date,
		stdout:  opt.Stdout,
		stderr:  opt.Stderr,
		ctx:     opt.Context,
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

func resolveBuildMeta(opt Opt) Opt {
	meta := runtimeBuildMeta()
	if needsVersionFallback(opt.Version) && meta.Version != "" {
		opt.Version = meta.Version
	}
	if needsCommitFallback(opt.Commit) && meta.Commit != "" {
		opt.Commit = meta.Commit
	}
	if needsDateFallback(opt.Date) && meta.Date != "" {
		opt.Date = meta.Date
	}
	return opt
}

type buildMeta struct {
	Version string
	Commit  string
	Date    string
}

func runtimeBuildMeta() buildMeta {
	info, ok := readBuildInfo()
	if !ok {
		return buildMeta{}
	}
	return buildMetaFromInfo(info)
}

func buildMetaFromInfo(info *debug.BuildInfo) buildMeta {
	if info == nil {
		return buildMeta{}
	}

	meta := buildMeta{}
	if version := trim(info.Main.Version); version != "" && version != "(devel)" {
		meta.Version = version
	}

	if revision := trim(buildSetting(info, "vcs.revision")); revision != "" {
		meta.Commit = shortRevision(revision)
		if buildSetting(info, "vcs.modified") == "true" {
			meta.Commit += "-dirty"
		}
	}

	meta.Date = formatBuildTime(buildSetting(info, "vcs.time"))
	return meta
}

func buildSetting(info *debug.BuildInfo, key string) string {
	for _, setting := range info.Settings {
		if setting.Key == key {
			return trim(setting.Value)
		}
	}
	return ""
}

func shortRevision(revision string) string {
	revision = trim(revision)
	if len(revision) <= 7 {
		return revision
	}
	return revision[:7]
}

func formatBuildTime(raw string) string {
	raw = trim(raw)
	if raw == "" {
		return ""
	}

	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return ts.UTC().Format("2006-01-02 15:04:05 MST")
}

func needsVersionFallback(version string) bool {
	version = trim(version)
	return version == "" || version == "dev" || version == "(devel)"
}

func needsCommitFallback(commit string) bool {
	commit = trim(commit)
	return commit == "" || commit == "unknown"
}

func needsDateFallback(date string) bool {
	date = trim(date)
	return date == "" || date == "unknown"
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
	c.fs.DurationVar(&c.runTimeout, "run-timeout", 0, "Whole-run timeout (0 disables)")
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
	if _, err := fmt.Fprintf(c.stderr, "Usage: %s [flags] [file]\n", c.use); err != nil {
		return
	}
	if _, err := fmt.Fprintln(c.stderr); err != nil {
		return
	}
	if _, err := fmt.Fprintln(c.stderr, "Flags:"); err != nil {
		return
	}
	c.fs.PrintDefaults()
}

func (c *cmd) run() error {
	if c.showVersion {
		return c.printVersion()
	}
	if c.runTimeout < 0 {
		return ExitErr{
			Err:  errors.New("run: --run-timeout must be >= 0"),
			Code: 2,
		}
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
	if isBlank(c.filePath) {
		return ExitErr{Err: errors.New("run: --file is required"), Code: 2}
	}

	targets, err := parseCompare(c.compareRaw)
	if err != nil {
		return ExitErr{
			Err:  fmt.Errorf("run: invalid --compare value: %w", err),
			Code: 2,
		}
	}

	ctx, cancel := c.runContext()
	defer cancel()

	rep, err := headless.Run(ctx, headless.Options{
		Version:       c.version,
		FilePath:      c.filePath,
		WorkspaceRoot: c.workspace,
		Recursive:     c.recursive,
		State: headless.StateOptions{
			ArtifactDir:    c.artifactDir,
			StateDir:       c.stateDir,
			PersistGlobals: c.persistVars,
			PersistAuth:    c.persistAuth,
			History:        c.history,
		},
		Environment: headless.EnvironmentOptions{
			Name:     c.envName,
			FilePath: c.envFile,
		},
		Compare: headless.CompareOptions{
			Targets: targets,
			Base:    c.compareBase,
		},
		Profile: c.profile,
		HTTP: headless.HTTPOptions{
			Timeout:            c.timeout,
			FollowRedirects:    boolPtr(c.follow),
			InsecureSkipVerify: c.insecure,
			ProxyURL:           c.proxyURL,
		},
		Selection: headless.Selection{
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

func (c *cmd) runContext() (context.Context, context.CancelFunc) {
	ctx := c.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	// Tie the root run context to process cancellation and an optional wall-clock deadline.
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	if c.runTimeout <= 0 {
		return ctx, stop
	}

	ctx, cancel := context.WithTimeout(ctx, c.runTimeout)
	return ctx, func() {
		cancel()
		stop()
	}
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
	path = trim(path)
	if path == "" {
		return nil
	}

	if merr := os.MkdirAll(filepath.Dir(path), 0o755); merr != nil {
		return merr
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

	sum, err := checksum()
	if err != nil {
		_, werr := fmt.Fprintf(c.stdout, "  sha256: unavailable (%v)\n", err)
		return werr
	}
	_, err = fmt.Fprintf(c.stdout, "  sha256: %s\n", sum)
	return err
}

func checksum() (string, error) {
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
	sum, err := checksumReader(f)
	if err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return sum, nil
}

func checksumReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func boolPtr(v bool) *bool {
	return &v
}
