package host

import (
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type Stats struct {
	Name      string
	CPUPct    float64
	GPUPct    float64
	MemPct    float64
	Cores     int
	Reachable bool
}

var gpuRe = regexp.MustCompile(`"Device Utilization %"=(\d+)`)

func gpu() float64 {
	out, err := exec.Command("ioreg", "-r", "-d", "1", "-w", "0", "-c", "IOAccelerator").Output()
	if err != nil {
		return -1
	}
	m := gpuRe.FindAllStringSubmatch(string(out), -1)
	if len(m) == 0 {
		return -1
	}
	best := 0
	for _, g := range m {
		if v, _ := strconv.Atoi(g[1]); v > best {
			best = v
		}
	}
	return float64(best)
}

func cpu(cores int) float64 {
	out, err := exec.Command("ps", "-A", "-o", "%cpu").Output()
	if err != nil {
		return -1
	}
	var sum float64
	for _, ln := range strings.Split(string(out), "\n")[1:] {
		v, err := strconv.ParseFloat(strings.TrimSpace(ln), 64)
		if err == nil {
			sum += v
		}
	}
	pct := sum / float64(cores)
	if pct > 100 {
		pct = 100
	}
	return pct
}

func mem() float64 {
	out, err := exec.Command("memory_pressure").Output()
	if err != nil {
		return -1
	}
	m := regexp.MustCompile(`(\d+)%`).FindStringSubmatch(string(out))
	if len(m) < 2 {
		return -1
	}
	v, _ := strconv.Atoi(m[1])
	return float64(100 - v)
}

func Local(name string) Stats {
	c := runtime.NumCPU()
	return Stats{Name: name, Cores: c, CPUPct: cpu(c), GPUPct: gpu(), MemPct: mem(), Reachable: true}
}

// Remote reads the other machine over ssh. Thunderbolt or wifi, same command —
// if the host is not configured or not up, it reports unreachable rather than zero.
func Remote(host, name string) Stats {
	cmd := exec.Command("ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=2", host,
		`echo "$(sysctl -n hw.ncpu)|$(ps -A -o %cpu | awk 'NR>1{s+=$1}END{print s}')|$(ioreg -r -d 1 -w 0 -c IOAccelerator 2>/dev/null | grep -o '"Device Utilization %"=[0-9]*' | head -1 | grep -o '[0-9]*$')"`)
	out, err := cmd.Output()
	if err != nil {
		return Stats{Name: name, Reachable: false}
	}
	p := strings.Split(strings.TrimSpace(string(out)), "|")
	if len(p) < 2 {
		return Stats{Name: name, Reachable: false}
	}
	cores, _ := strconv.Atoi(p[0])
	total, _ := strconv.ParseFloat(p[1], 64)
	g := -1.0
	if len(p) > 2 && p[2] != "" {
		if v, e := strconv.Atoi(p[2]); e == nil {
			g = float64(v)
		}
	}
	pct := 0.0
	if cores > 0 {
		pct = total / float64(cores)
	}
	if pct > 100 {
		pct = 100
	}
	return Stats{Name: name, Cores: cores, CPUPct: pct, GPUPct: g, Reachable: true}
}
