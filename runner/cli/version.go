package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"time"
)

// Kept as a variable so tests can replace it.
var readBuildInfo = debug.ReadBuildInfo

// Keep runtime build dates consistent with release metadata.
const buildTimeLayout = "2006-01-02 15:04:05 MST"

type buildMeta struct {
	Version string
	Commit  string
	Date    string
}

// Explicit linker values take precedence over runtime build info.
func resolveBuildMeta(opt Opt) Opt {
	meta := runtimeBuildMeta()
	opt.Version = metaOr(opt.Version, meta.Version, "dev", "(devel)")
	opt.Commit = metaOr(opt.Commit, meta.Commit, "unknown")
	opt.Date = metaOr(opt.Date, meta.Date, "unknown")
	return opt
}

func metaOr(configured, discovered string, placeholders ...string) string {
	if discovered != "" && (configured == "" || slices.Contains(placeholders, configured)) {
		return discovered
	}
	return configured
}

// Installed binaries carry a module version; local builds use VCS data.
func runtimeBuildMeta() buildMeta {
	info, ok := readBuildInfo()
	if !ok || info == nil {
		return buildMeta{}
	}

	var meta buildMeta
	if version := strings.TrimSpace(info.Main.Version); version != "(devel)" {
		meta.Version = version
	}
	if revision := buildSetting(info, "vcs.revision"); revision != "" {
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
			return strings.TrimSpace(setting.Value)
		}
	}
	return ""
}

func shortRevision(revision string) string {
	if len(revision) <= 7 {
		return revision
	}
	return revision[:7]
}

func formatBuildTime(raw string) string {
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	return ts.UTC().Format(buildTimeLayout)
}

func (c *cmd) printVersion() error {
	sum, err := checksum()
	if err != nil {
		sum = fmt.Sprintf("unavailable (%v)", err)
	}

	_, err = fmt.Fprintf(c.Stdout, "%s %s\n  commit: %s\n  built:  %s\n  sha256: %s\n",
		c.Use, c.Version, c.Commit, c.Date, sum)
	return err
}

// checksum returns the SHA-256 hash of the running binary.
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
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
