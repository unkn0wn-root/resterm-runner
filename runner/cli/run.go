package cli

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/unkn0wn-root/resterm/headless"
)

const defaultUse = "resterm-runner"

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
	Opt
	fs *flag.FlagSet

	filePath  string
	workspace string
	recursive bool
	reqName   string
	workflow  string
	tag       string
	all       bool

	envName   string
	envGroups groupFlags
	envFile   string

	compareRaw   string
	compareBase  string
	compareGroup string

	timeout  time.Duration
	insecure bool
	follow   bool
	proxyURL string

	artifactDir string
	reportJSON  string
	reportJUnit string
	stateDir    string
	persistAuth bool
	persistVars bool
	history     bool

	runTimeout   time.Duration
	exitCodeMode string
	profile      bool
	failFast     bool
	showVersion  bool
}

func Run(args []string, opt Opt) error {
	if opt.Stdout == nil || opt.Stderr == nil {
		return headless.ErrNilWriter
	}

	c := newCmd(opt)
	if err := c.fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return usageErr("%w", err)
	}
	return c.run()
}

func newCmd(opt Opt) *cmd {
	opt.Use = cmp.Or(strings.TrimSpace(opt.Use), defaultUse)
	opt.Version = strings.TrimSpace(opt.Version)
	opt.Commit = strings.TrimSpace(opt.Commit)
	opt.Date = strings.TrimSpace(opt.Date)

	c := &cmd{Opt: resolveBuildMeta(opt)}
	c.fs = flag.NewFlagSet(c.Use, flag.ContinueOnError)
	c.fs.SetOutput(c.Stderr)
	c.fs.Usage = c.usage
	c.bind()
	return c
}

func (c *cmd) bind() {
	c.fs.StringVar(&c.filePath, "file", "", "Path to .http/.rest file to run")
	c.fs.StringVar(&c.workspace, "workspace", "", "Workspace directory to scan for request files")
	c.fs.BoolVar(&c.recursive, "recursive", false, "Recursively scan workspace for request files")
	c.fs.StringVar(&c.reqName, "request", "", "Request name to run")
	c.fs.StringVar(&c.workflow, "workflow", "", "Workflow name to run")
	c.fs.StringVar(&c.tag, "tag", "", "Run requests with the given tag")
	c.fs.BoolVar(&c.all, "all", false, "Run all requests in the file")

	c.fs.StringVar(&c.envName, "env", "", "Environment name to use")
	c.fs.Var(&c.envGroups, "env-group", "Select a grouped environment as group=profile (repeatable)")
	c.fs.StringVar(&c.envFile, "env-file", "", "Path to environment file")

	c.fs.StringVar(&c.compareRaw, "compare", "", "Compare environments (comma/space separated)")
	c.fs.StringVar(&c.compareBase, "compare-base", "", "Baseline environment for --compare")
	c.fs.StringVar(&c.compareGroup, "compare-group", "", "Environment group varied by --compare")

	c.fs.DurationVar(&c.timeout, "timeout", 30*time.Second, "Request timeout")
	c.fs.BoolVar(&c.insecure, "insecure", false, "Skip TLS certificate verification")
	c.fs.BoolVar(&c.follow, "follow", true, "Follow redirects")
	c.fs.StringVar(&c.proxyURL, "proxy", "", "HTTP proxy URL")

	c.fs.StringVar(&c.artifactDir, "artifact-dir", "", "Write run artifacts to the given directory")
	c.fs.StringVar(&c.reportJSON, "report-json", "", "Write JSON report to the given path")
	c.fs.StringVar(&c.reportJUnit, "report-junit", "", "Write JUnit XML report to the given path")
	c.fs.StringVar(&c.stateDir, "state-dir", "", "Directory for persisted runner state")
	c.fs.BoolVar(&c.persistVars, "persist-globals", false, "Persist runtime globals and file vars between runs")
	c.fs.BoolVar(&c.persistAuth, "persist-auth", false, "Persist auth caches between runs")
	c.fs.BoolVar(&c.history, "history", false, "Persist request history for runner executions")

	c.fs.DurationVar(&c.runTimeout, "run-timeout", 0, "Whole-run timeout (0 disables)")
	c.fs.StringVar(&c.exitCodeMode, "exit-code-mode", string(headless.ExitCodeDetailed), "Exit code mode: detailed or summary")
	c.fs.BoolVar(&c.profile, "profile", false, "Profile the selected request run(s)")
	c.fs.BoolVar(&c.failFast, "fail-fast", false, "Stop after the first failed result")
	c.fs.BoolVar(&c.showVersion, "version", false, "Show resterm-runner version")
}

func (c *cmd) usage() {
	_, _ = fmt.Fprintf(c.Stderr, "Usage: %s [flags] [file]\n\nFlags:\n", c.Use)
	c.fs.PrintDefaults()
}

func (c *cmd) run() error {
	if c.showVersion {
		return c.printVersion()
	}
	if c.runTimeout < 0 {
		return usageErr("run: --run-timeout must be >= 0")
	}

	rest := c.fs.Args()
	if c.filePath == "" && len(rest) > 0 {
		c.filePath = rest[0]
	}
	if len(rest) > 1 {
		return usageErr("run: unexpected args: %s", strings.Join(rest[1:], " "))
	}
	if strings.TrimSpace(c.filePath) == "" {
		return usageErr("run: --file is required")
	}

	compareTargets, err := parseCompare(c.compareRaw)
	if err != nil {
		return usageErr("run: invalid --compare value: %w", err)
	}
	exitCodeMode, err := parseExitCodeMode(c.exitCodeMode)
	if err != nil {
		return usageErr("run: %w", err)
	}

	ctx, cancel := c.runContext()
	defer cancel()

	pl, err := headless.Build(c.options(compareTargets))
	if err != nil {
		return runErr(err)
	}
	rep, err := headless.RunPlan(ctx, pl)
	if err != nil {
		return runErr(err)
	}

	if err := rep.WriteText(c.Stdout); err != nil {
		return fmt.Errorf("run: write output: %w", err)
	}
	if err := writeReport(c.reportJSON, rep.WriteJSON); err != nil {
		return fmt.Errorf("run: write json report: %w", err)
	}
	if err := writeReport(c.reportJUnit, rep.WriteJUnit); err != nil {
		return fmt.Errorf("run: write junit report: %w", err)
	}

	if rep.HasFailures() {
		return ExitErr{
			Err:  errors.New("one or more requests failed"),
			Code: rep.ExitCode(exitCodeMode),
		}
	}
	return nil
}

func (c *cmd) options(compareTargets []string) headless.Options {
	return headless.Options{
		Version:       c.Version,
		Source:        headless.Source{Path: c.filePath},
		WorkspaceRoot: c.workspace,
		Recursive:     c.recursive,
		FailFast:      c.failFast,
		State: headless.StateOptions{
			ArtifactDir:    c.artifactDir,
			StateDir:       c.stateDir,
			PersistGlobals: c.persistVars,
			PersistAuth:    c.persistAuth,
			History:        c.history,
		},
		Environment: headless.EnvironmentOptions{
			Name:      c.envName,
			Selection: headless.EnvironmentSelection(c.envGroups),
			FilePath:  c.envFile,
		},
		Compare: headless.CompareOptions{
			Targets: compareTargets,
			Base:    c.compareBase,
			Group:   c.compareGroup,
		},
		Profile: headless.ProfileOptions{
			Enabled: c.profile,
		},
		HTTP: headless.HTTPOptions{
			Timeout:            c.timeout,
			FollowRedirects:    &c.follow,
			InsecureSkipVerify: c.insecure,
			ProxyURL:           c.proxyURL,
		},
		Selection: headless.Selection{
			Request:  c.reqName,
			Workflow: c.workflow,
			Tag:      c.tag,
			All:      c.all,
		},
	}
}

func (c *cmd) runContext() (context.Context, context.CancelFunc) {
	ctx := c.Context
	if ctx == nil {
		ctx = context.Background()
	}

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

func parseExitCodeMode(raw string) (headless.ExitCodeMode, error) {
	switch mode := headless.ExitCodeMode(strings.ToLower(strings.TrimSpace(raw))); mode {
	case "", headless.ExitCodeDetailed:
		return headless.ExitCodeDetailed, nil
	case headless.ExitCodeSummary:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported --exit-code-mode %q", raw)
	}
}

func writeReport(path string, write func(io.Writer) error) (err error) {
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
	return write(f)
}
