package collect

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// sessionIDRe is the UUID Claude Code uses as a session file name.
var sessionIDRe = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// ClaudeLive is the set of session IDs and process working directories that
// belong to a running Claude Code CLI. Keys are matched in ClaudeSessions.
// Claude Desktop / Cowork are not listed: one app process is not one session,
// and its cwd is not a project.
func ClaudeLive() map[string]bool {
	live := map[string]bool{}
	out, err := exec.Command("ps", "-eo", "pid=,args=").Output()
	if err != nil {
		return live
	}
	for _, ln := range strings.Split(string(out), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" {
			continue
		}
		pid, args, ok := splitPIDArgs(ln)
		if !ok || !isClaudeCLI(args) {
			continue
		}
		if cwd := procCWD(pid); cwd != "" {
			live[cwd] = true
		}
		if id := resumeSessionID(args); id != "" {
			live[id] = true
		}
	}
	return live
}

func splitPIDArgs(ln string) (pid, args string, ok bool) {
	i := 0
	for i < len(ln) && ln[i] >= '0' && ln[i] <= '9' {
		i++
	}
	if i == 0 {
		return "", "", false
	}
	return ln[:i], strings.TrimSpace(ln[i:]), true
}

func isClaudeCLI(args string) bool {
	f := strings.Fields(args)
	if len(f) == 0 {
		return false
	}
	base := filepath.Base(f[0])
	if base == "claude" {
		return true
	}
	// node /path/@anthropic-ai/claude-code/cli.js …
	if strings.Contains(args, "claude-code") && strings.Contains(args, "cli.js") {
		return true
	}
	return false
}

func resumeSessionID(args string) string {
	f := strings.Fields(args)
	for i, a := range f {
		if a != "--resume" && a != "-r" {
			continue
		}
		if i+1 < len(f) && sessionIDRe.MatchString(f[i+1]) {
			return f[i+1]
		}
	}
	return ""
}

func procCWD(pid string) string {
	if pid == "" {
		return ""
	}
	if runtime.GOOS == "linux" {
		if p, err := os.Readlink("/proc/" + pid + "/cwd"); err == nil {
			return p
		}
	}
	out, err := exec.Command("lsof", "-a", "-p", pid, "-d", "cwd", "-Fn").Output()
	if err != nil {
		return ""
	}
	for _, ln := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(ln, "n") {
			return strings.TrimPrefix(ln, "n")
		}
	}
	return ""
}
