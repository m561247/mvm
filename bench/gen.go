// bench/gen.go runs the cross-language microbenchmarks via hyperfine,
// assembles bench/bench.json, and refreshes the <!-- bench:start --> marker
// block in the top-level README.md. Invoked by `make bench`.
//
//go:build ignore

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

const schemaVersion = 1

type result struct {
	Runtime  string  `json:"runtime"`
	Version  string  `json:"version"`
	MeanMs   float64 `json:"meanMs"`
	StddevMs float64 `json:"stddevMs"`
}

type bench struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Results     []result `json:"results"`
}

type matrix struct {
	SchemaVersion int     `json:"schemaVersion"`
	GeneratedAt   string  `json:"generatedAt"`
	Mvm           string  `json:"mvm"`
	Platform      string  `json:"platform"`
	CPU           string  `json:"cpu"`
	Note          string  `json:"note"`
	Benchmarks    []bench `json:"benchmarks"`
}

type cmd struct {
	runtime string
	bin     string
	args    []string
}

type workload struct {
	name        string
	description string
	cmds        []cmd
}

// hyperfineOut is the subset of hyperfine --export-json we read.
type hyperfineOut struct {
	Results []struct {
		Command string  `json:"command"`
		Mean    float64 `json:"mean"`   // seconds
		Stddev  float64 `json:"stddev"` // seconds
	} `json:"results"`
}

func main() {
	var (
		mvmBin     = flag.String("mvm", "./mvm", "path to mvm binary")
		luaBin     = flag.String("lua5.4", "lua5.4", "path to lua 5.4 binary")
		lua51Bin   = flag.String("lua5.1", "lua5.1", "path to lua 5.1 binary")
		pyBin      = flag.String("python", "python3", "path to python3 binary")
		out        = flag.String("o", "bench/bench.json", "output JSON path")
		readmePath = flag.String("readme", "README.md", "path to README.md to refresh")
		runs       = flag.Int("runs", 5, "min runs per command (hyperfine --min-runs)")
		warmup     = flag.Int("warmup", 1, "warmup runs (hyperfine --warmup)")
		dir        = flag.String("dir", "bench", "directory containing the workload scripts")
	)
	flag.Parse()

	if _, err := exec.LookPath("hyperfine"); err != nil {
		die("hyperfine not found in PATH; install from https://github.com/sharkdp/hyperfine")
	}

	mvmAbs, err := filepath.Abs(*mvmBin)
	if err != nil {
		die("resolve mvm path: %v", err)
	}
	if _, err := os.Stat(mvmAbs); err != nil {
		die("mvm binary not found at %s (build it first: go build -o mvm .)", mvmAbs)
	}

	mvmVersion := strings.TrimSpace(mustRun("git", "rev-parse", "--short", "HEAD"))
	if mvmVersion == "" {
		mvmVersion = "unknown"
	}

	wls := []workload{
		{
			name:        "sieve",
			description: "Eratosthenes sieve, N=10_000_000",
			cmds: []cmd{
				{"mvm", mvmAbs, []string{"run", "sieve.go"}},
				{"lua5.4", *luaBin, []string{"sieve.lua"}},
				{"lua5.1", *lua51Bin, []string{"sieve.lua"}},
				{"python3", *pyBin, []string{"sieve.py"}},
			},
		},
		{
			name:        "fib35",
			description: "Recursive fib(35)",
			cmds: []cmd{
				{"mvm", mvmAbs, []string{"run", "fib35.go"}},
				{"lua5.4", *luaBin, []string{"fib35.lua"}},
				{"lua5.1", *lua51Bin, []string{"fib35.lua"}},
				{"python3", *pyBin, []string{"fib35.py"}},
			},
		},
	}

	m := matrix{
		SchemaVersion: schemaVersion,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Mvm:           mvmVersion,
		Platform:      runtime.GOOS + "/" + runtime.GOARCH,
		CPU:           detectCPU(),
		Note:          "Single-process wall time via hyperfine (--warmup 1 --min-runs 5). Numbers vary by hardware; reproduce locally with `make bench`.",
	}

	for _, w := range wls {
		fmt.Fprintf(os.Stderr, "==> %s\n", w.name)
		b := bench{Name: w.name, Description: w.description}
		for _, c := range w.cmds {
			r, err := runOne(*dir, c, *runs, *warmup)
			if err != nil {
				fmt.Fprintf(os.Stderr, "  %s: %v (skipped)\n", c.runtime, err)
				continue
			}
			r.Version = detectVersion(c)
			b.Results = append(b.Results, r)
			fmt.Fprintf(os.Stderr, "  %-8s %s  %7.1f ms +/- %.1f\n",
				c.runtime, r.Version, r.MeanMs, r.StddevMs)
		}
		sort.SliceStable(b.Results, func(i, j int) bool { return b.Results[i].MeanMs < b.Results[j].MeanMs })
		m.Benchmarks = append(m.Benchmarks, b)
	}

	buf, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		die("marshal: %v", err)
	}
	if err := os.WriteFile(*out, append(buf, '\n'), 0o644); err != nil { //nolint:gosec
		die("write %s: %v", *out, err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", *out)

	if err := updateReadme(*readmePath, m); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
}

func runOne(dir string, c cmd, runs, warmup int) (result, error) {
	if _, err := exec.LookPath(c.bin); err != nil {
		return result{}, fmt.Errorf("%s not in PATH", c.bin)
	}
	tmp, err := os.CreateTemp("", "bench-*.json")
	if err != nil {
		return result{}, err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	// hyperfine treats each positional arg as one command to benchmark, so
	// flatten (bin, args...) into a single shell-style string.
	cmdStr := c.bin
	for _, a := range c.args {
		cmdStr += " " + a
	}
	args := []string{
		"--warmup", fmt.Sprint(warmup),
		"--min-runs", fmt.Sprint(runs),
		"--export-json", tmp.Name(),
		"-N", // --shell=none: run the command directly, no shell wrapper
		cmdStr,
	}

	cmd := exec.Command("hyperfine", args...)
	cmd.Dir = dir
	var stderr strings.Builder
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return result{}, fmt.Errorf("hyperfine %q: %v: %s", cmdStr, err, strings.TrimSpace(stderr.String()))
	}

	buf, err := os.ReadFile(tmp.Name())
	if err != nil {
		return result{}, err
	}
	var h hyperfineOut
	if err := json.Unmarshal(buf, &h); err != nil {
		return result{}, err
	}
	if len(h.Results) == 0 {
		return result{}, errors.New("hyperfine produced no results")
	}
	r := h.Results[0]
	return result{
		Runtime:  c.runtime,
		MeanMs:   round1(r.Mean * 1000),
		StddevMs: round1(r.Stddev * 1000),
	}, nil
}

func detectVersion(c cmd) string {
	switch c.runtime {
	case "mvm":
		return strings.TrimSpace(mustRun("git", "rev-parse", "--short", "HEAD"))
	case "lua5.4", "lua5.1":
		out, _ := exec.Command(c.bin, "-v").CombinedOutput()
		// "Lua 5.4.8  Copyright ..." -> "5.4.8"
		fields := strings.Fields(string(out))
		if len(fields) >= 2 {
			return fields[1]
		}
		return ""
	case "python3":
		out, _ := exec.Command(c.bin, "--version").Output()
		// "Python 3.12.13" -> "3.12.13"
		fields := strings.Fields(string(out))
		if len(fields) >= 2 {
			return fields[1]
		}
		return ""
	}
	return ""
}

func detectCPU() string {
	// Best-effort across Linux flavors and macOS.
	if buf, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for line := range strings.SplitSeq(string(buf), "\n") {
			for _, key := range []string{"model name", "Hardware", "Processor"} {
				if strings.HasPrefix(line, key) {
					if i := strings.Index(line, ":"); i >= 0 {
						return strings.TrimSpace(line[i+1:])
					}
				}
			}
		}
	}
	if out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
		s := strings.TrimSpace(string(out))
		if s != "" {
			return s
		}
	}
	return runtime.GOARCH
}

func mustRun(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

func round1(f float64) float64 {
	return float64(int(f*10+0.5)) / 10
}

var reReadme = regexp.MustCompile(`(?s)<!-- bench:start -->.*?<!-- bench:end -->`)

func updateReadme(path string, m matrix) error {
	buf, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if !reReadme.Match(buf) {
		return errors.New("markers <!-- bench:start --> / <!-- bench:end --> not found in README")
	}
	// Headline: sieve numbers, sorted ascending (best first).
	var line string
	for _, b := range m.Benchmarks {
		if b.Name != "sieve" {
			continue
		}
		var parts []string
		for _, r := range b.Results {
			parts = append(parts, fmt.Sprintf("%s %.0f ms", r.Runtime, r.MeanMs))
		}
		line = "sieve(10M): " + strings.Join(parts, ", ")
		break
	}
	if line == "" && len(m.Benchmarks) > 0 {
		line = m.Benchmarks[0].Name + ": see bench.json"
	}
	date := m.GeneratedAt
	if len(date) >= 10 {
		date = date[:10]
	}
	block := fmt.Sprintf("<!-- bench:start -->\n%s (%s, %s).\nSee the full table at https://mvm.sh/bench.\n<!-- bench:end -->",
		line, m.Platform, date)
	out := reReadme.ReplaceAllLiteralString(string(buf), block)
	return os.WriteFile(path, []byte(out), 0o644) //nolint:gosec
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "bench/gen: "+format+"\n", args...)
	os.Exit(1)
}
