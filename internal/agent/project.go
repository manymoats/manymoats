package agent

import "hash/fnv"

// Projects get a colour so a folder can be recognised without reading its name.
// Deliberately distinct from every provider hue in marks.json — a project is not
// an agent, and the two must never be mistaken for each other.
var projectHues = []string{
	"#7DD3FC", "#FDBA74", "#86EFAC", "#F0ABFC", "#FCA5A5",
	"#A5B4FC", "#FDE68A", "#5EEAD4", "#D8B4FE", "#BEF264",
}

func ProjectHue(p string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(p))
	return projectHues[int(h.Sum32())%len(projectHues)]
}

// The folder emblem sits WITH its agent, not in a header above a group — you
// should see which project an agent belongs to without moving your eye. Three
// cells so it reads as a band rather than a speck.
const FolderMark = "███"

// FolderTab is the one-cell form, for legends and tight strips.
const FolderTab = "█"

// GroupByProject returns projects in a stable order with their agents. When
// everything is in one project the caller should not label anything — repeating
// the same folder on every row is noise, not information.
func GroupByProject(as []Agent) (order []string, byProject map[string][]Agent) {
	byProject = map[string][]Agent{}
	for _, a := range as {
		p := a.Project
		if p == "" {
			p = UnknownProject
		}
		if _, seen := byProject[p]; !seen {
			order = append(order, p)
		}
		byProject[p] = append(byProject[p], a)
	}
	return order, byProject
}

func SingleProject(as []Agent) bool {
	order, _ := GroupByProject(as)
	return len(order) <= 1
}
