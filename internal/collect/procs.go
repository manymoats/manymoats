package collect

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/manymoats/manymoats/internal/agent"
)

var cursorWS = regexp.MustCompile(`extension-host\s+([^\s\[]+)`)

func Processes() []agent.Agent {
	out, err := exec.Command("ps", "-eo", "pid,pcpu,etime,args").Output()
	if err != nil {
		return nil
	}
	// Cursor spreads one agent's work across an extension host AND its renderers;
	// the host alone can read near-idle while the renderers stream a response.
	// Total Cursor CPU is the honest signal.
	var cursorCPU float64
	for _, ln := range strings.Split(string(out), "\n")[1:] {
		f := strings.Fields(ln)
		if len(f) < 4 || !strings.Contains(strings.Join(f[3:], " "), "Cursor") {
			continue
		}
		c, _ := strconv.ParseFloat(f[1], 64)
		cursorCPU += c
	}

	var res []agent.Agent
	seen := map[string]bool{}
	for _, ln := range strings.Split(string(out), "\n")[1:] {
		f := strings.Fields(ln)
		if len(f) < 4 {
			continue
		}
		cpu, _ := strconv.ParseFloat(f[1], 64)
		args := strings.Join(f[3:], " ")

		switch {
		case false: // superseded by collect.Cursor(), which reads Cursor's own chat records
			mm := cursorWS.FindStringSubmatch(args)
			proj := agent.UnknownProject
			if len(mm) > 1 && mm[1] != "Agents" {
				proj = mm[1]
			}
			key := "cursor/" + proj
			if seen[key] {
				continue
			}
			seen[key] = true
			st := agent.Idle
			// Cursor's own share of the machine, summed across its processes. It is
			// the only activity number Cursor exposes — no tokens, no rate.
			// Two numbers, two jobs. WHETHER Cursor is working comes from its
			// total share of the machine — the renderers stream the response, so
			// the host alone reads near-idle while an agent is mid-answer. WHAT
			// this row shows is that host's own CPU, which is the only part
			// actually attributable to this chat.
			// ps reports %cpu as a share of ONE core; that is the standard unit and
			// what a person expects beside a process name. Machine bars use % of
			// the whole machine — different question, different unit, both labelled.
			share := cpu
			// WHETHER it is working comes from Cursor's total, because the renderers
			// carry the streaming work while the host reads near-idle.
			if cursorCPU >= 12 {
				st = agent.Working
			}
			res = append(res, agent.Agent{
				ID: f[0], Source: agent.Cursor, Model: "cursor", Project: proj,
				State: st, Since: parseETime(f[2]), CPUPct: share,
			})
		case strings.Contains(args, "grok") && strings.Contains(args, "-p"):
			if seen["grok"] {
				continue
			}
			seen["grok"] = true
			res = append(res, agent.Agent{
				ID: f[0], Source: agent.Grok, Model: "grok", Project: agent.UnknownProject,
				State: agent.Working, Since: parseETime(f[2]),
			})
		}
	}
	return res
}

func parseETime(s string) time.Duration {
	var d time.Duration
	if i := strings.Index(s, "-"); i >= 0 {
		days, _ := strconv.Atoi(s[:i])
		d += time.Duration(days) * 24 * time.Hour
		s = s[i+1:]
	}
	p := strings.Split(s, ":")
	mult := []time.Duration{time.Second, time.Minute, time.Hour}
	for i := 0; i < len(p) && i < 3; i++ {
		v, _ := strconv.Atoi(p[len(p)-1-i])
		d += time.Duration(v) * mult[i]
	}
	return d
}
