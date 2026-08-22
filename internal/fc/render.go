package fc

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/manymoats/manymoats/internal/credits"
)

type app struct {
	cat      *credits.Catalog
	bals     map[string]credits.Balance
	held     map[string]bool
	holdings bool
	m        marks
	now      time.Time
}

// money never prints a bare number. A figure without its unit is a lie by
// omission, and this one is dollars.
func money(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	whole, frac, _ := strings.Cut(s, ".")
	neg := strings.HasPrefix(whole, "-")
	whole = strings.TrimPrefix(whole, "-")
	var parts []string
	for len(whole) > 3 {
		parts = append([]string{whole[len(whole)-3:]}, parts...)
		whole = whole[:len(whole)-3]
	}
	parts = append([]string{whole}, parts...)
	out := "$" + strings.Join(parts, ",") + "." + frac
	if neg {
		return "-" + out
	}
	return out
}

// ageLabel is the whole reason the declared tier exists. It never prints a zero
// and it never leaves a declared figure undated.
func ageLabel(b credits.Balance) string {
	if b.Source == credits.Live {
		return "asked just now"
	}
	if b.Amount == nil {
		return "never checked"
	}
	switch d := b.AgeDays; {
	case d < 0:
		return "age unknown"
	case d == 0:
		return "today"
	case d == 1:
		return "1 day old"
	default:
		return fmt.Sprint(d) + " days old"
	}
}

// leftLabel enforces rule one in the printer as well as the library: a number
// nobody can date is not shown as a number at all.
func leftLabel(c credits.Credit, b credits.Balance) string {
	if b.Amount == nil {
		return string(credits.Unknown)
	}
	if b.Source != credits.Live && b.AgeDays < 0 {
		return string(credits.Unknown)
	}
	if c.Unit != "" && c.Unit != "usd" {
		return fmt.Sprintf("%.0f %s", *b.Amount, c.Unit)
	}
	return money(*b.Amount)
}

func (a app) frame() frame { return frame{width: frameW, ascii: a.m.plain} }

func (a app) balance(id string) credits.Balance {
	if b, ok := a.bals[id]; ok {
		return b
	}
	return credits.Balance{CreditID: id, Source: credits.Declared}
}

func (a app) shown() []credits.Credit {
	var out []credits.Credit
	for _, c := range a.cat.Credits {
		if a.holdings && !a.held[c.ID] {
			continue
		}
		out = append(out, c)
	}
	return out
}

// ── surface: your credits ──────────────────────────────────────────────────

func (a app) creditsView() string {
	f := a.frame()
	var b strings.Builder
	p := func(s string) { b.WriteString(s + "\n") }

	p(f.top())
	p(f.prose("your credits"))
	p(f.prose("the one dying soonest is first"))
	p(f.line(""))
	p(a.m.headRow(f))
	p(f.line(""))

	list := a.shown()
	for _, c := range list {
		bal := a.balance(c.ID)
		c.Balance, c.HasBal = 0, false
		if bal.Amount != nil {
			c.Balance, c.HasBal = *bal.Amount, true
		}
		p(a.m.row(f, a.m.of(bal.Source), c.Name, hue(c.Colour, a.m.plain),
			leftLabel(c, bal), c.ExpiryLabel(), ageLabel(bal)))
		for _, d := range a.details(c, bal) {
			for _, l := range fitLines(d, f.detailW()) {
				p(f.detail(l))
			}
		}
		p(f.line(""))
	}

	if len(list) == 0 {
		p(f.detail("no credits to show"))
		p(f.line(""))
	}

	p(f.mid())
	if a.m.plain {
		p(f.prose("live   we asked the provider just now"))
		p(f.prose("part   a starting figure minus what we measured you spending"))
		p(f.prose("old    written down by hand — we always say how old it is"))
	} else {
		p(f.prose("● we asked the provider just now   ◐ a starting figure minus what"))
		p(f.prose("○ written down by hand — we always    we measured you spending"))
		p(f.prose("  say how old it is"))
	}
	p(f.line(""))
	p(f.prose("nothing here is a guess. what we cannot check says unknown."))
	if !a.holdings {
		p(f.prose("we do not know which of these you hold. tell us in"))
		p(f.prose("~/.manymoats/credits/holdings.json and this becomes your list."))
	}
	p(f.bot())
	return b.String()
}

// details are the plain-words lines under a credit. Each one earns its row: no
// zeroes, no blanks the reader will fill in optimistically.
func (a app) details(c credits.Credit, bal credits.Balance) []string {
	var out []string
	if c.Burning != "" {
		out = append(out, "now being spent by  "+c.Burning)
	}
	if pd, ok := c.PerDay(); ok && pd >= 0.01 {
		out = append(out, "spend "+money(pd)+" a day to use it before it dies")
	}
	if bal.Detail != "" {
		out = append(out, bal.Detail)
	}
	if bal.Amount == nil && bal.Err != nil {
		out = append(out, whyUnknown(c, bal))
	}
	return out
}

func whyUnknown(c credits.Credit, bal credits.Balance) string {
	if errors.Is(bal.Err, credits.ErrNoNetwork) {
		return "we did not ask — --no-network was set"
	}
	switch c.Source.Method {
	case "console_only":
		where := c.Source.Where
		if where == "" {
			where = "the provider's own billing page"
		}
		return "nobody can check this one — it's only on " + where
	case "api":
		return "we asked and got no answer: " + bal.Err.Error()
	}
	return "no working balance check has ever been proven for this one"
}

// ── surface: what a credit actually pays for ───────────────────────────────

const doorDef = "A door is the web address you send the request to. The same model often " +
	"has two, and they don't bill the same."

func (a app) coversView(model string) string {
	var b strings.Builder
	p := func(s string) { b.WriteString(s + "\n") }
	p("")
	p("  " + invocation + " covers " + model)
	p("")
	for _, l := range wrap(doorDef, 74) {
		p("  " + l)
	}
	p("")

	// The credit that answers two ways about the same model is the one that can
	// cost money, so it goes first. Nothing else here is ranked.
	type block struct {
		c   credits.Credit
		cov []credits.Coverage
	}
	var blocks []block
	for _, c := range a.cat.Credits {
		if cov := a.cat.DoorsFor(c.ID, model); len(cov) > 0 {
			blocks = append(blocks, block{c, cov})
		}
	}
	sort.SliceStable(blocks, func(i, j int) bool {
		return a.cat.Disagrees(blocks[i].cov) && !a.cat.Disagrees(blocks[j].cov)
	})

	if len(blocks) == 0 {
		p("  " + rule(76))
		p("")
		p("  " + string(credits.Unknown))
		p("")
		for _, l := range wrap("No credit we know of has an answer for "+model+
			" at any door. We print unknown rather than a guess.", 74) {
			p("  " + l)
		}
		p("")
		p("  The command that would settle it by asking the door directly,")
		p("  " + invocation + " check, is not built yet.")
		p("")
		return b.String()
	}

	for _, bl := range blocks {
		p("  " + rule(76))
		p("")
		b.WriteString(a.creditCovers(bl.c, model, bl.cov))
	}
	p("  " + rule(76))
	p("")
	p("  Any door not listed above is unknown. We have not checked it, and we")
	p("  do not assume a credit covers something because the provider matches.")
	p("")
	p("  " + invocation + " show <credit>  is one credit in full, every door we know.")
	p("")
	return b.String()
}

// creditCovers is the door-by-door answer for ONE model. The whole rule set of
// a credit belongs to `show`; putting it here made the answer to a question
// three screens long and hid the two-door case inside it.
func (a app) creditCovers(c credits.Credit, model string, cov []credits.Coverage) string {
	var b strings.Builder
	p := func(s string) { b.WriteString(s + "\n") }

	bal := a.balance(c.ID)
	p("  paid by  " + c.Name + "  ·  " + leftLabel(c, bal) + " left  ·  " + c.ExpiryLabel())
	p("")

	var yes, no *credits.Coverage
	for i := range cov {
		switch cov[i].Verdict {
		case credits.Covered:
			if yes == nil {
				yes = &cov[i]
			}
		case credits.NotCovered:
			if no == nil {
				no = &cov[i]
			}
		}
	}
	// The two-door answer is the product. It is drawn only when the same model
	// really does resolve both ways on one credit.
	if yes != nil && no != nil {
		for _, l := range a.twoDoors(model, *yes, *no) {
			p(l)
		}
		p("")
		p("        same model  ·  two doors  ·  only one is on your credit")
		p("")
		p("  What the second door costs depends on your own billing setup — we")
		p("  cannot see that. We only know this credit does not cover it.")
		p("")
	}

	p("  " + pad("verdict", verdictW) + pad("what", whatW) + "through which door")
	p("")
	for _, cv := range cov {
		for _, l := range tableRow(verdictWord(cv.Verdict), covWhat(cv, model), a.cat.DoorName(cv.DoorID)) {
			p(l)
		}
	}
	p("")

	// When every door was checked on the same day off the same page, saying so
	// once is the honest version. Repeating it per door made the line overrun
	// the frame and told the reader nothing extra.
	for _, g := range groupProvenance(cov) {
		p("  " + checked(g.cov[0]) + doorSuffix(a, g.cov, cov))
		if src := g.cov[0].Source; src != "" {
			p("  " + strings.TrimPrefix(strings.TrimPrefix(src, "https://"), "http://"))
		}
		if g.cov[0].Stale {
			p("  this may have changed — check the source above")
		}
		p("")
	}
	for _, cv := range cov {
		if cv.Withdrawn {
			for _, l := range a.withdrawn(c, cv) {
				p(l)
			}
		}
	}
	if c.Trap != "" {
		p("  worth knowing")
		p("")
		for _, l := range wrap(c.Trap, 72) {
			p("    " + l)
		}
		p("")
	}
	return b.String()
}

type provGroup struct {
	key string
	cov []credits.Coverage
}

func groupProvenance(cov []credits.Coverage) []provGroup {
	var out []provGroup
	for _, cv := range cov {
		k := fmt.Sprintf("%s|%d|%s|%t", cv.Source, cv.AgeDays, cv.Clock, cv.Stale)
		found := false
		for i := range out {
			if out[i].key == k {
				out[i].cov = append(out[i].cov, cv)
				found = true
				break
			}
		}
		if !found {
			out = append(out, provGroup{k, []credits.Coverage{cv}})
		}
	}
	return out
}

// doorSuffix names the door only when naming it adds something, and only when
// it fits. A line that wraps in an 80-column terminal is a defect, not a detail.
func doorSuffix(a app, group, all []credits.Coverage) string {
	if len(group) == len(all) {
		return ""
	}
	s := ", for " + a.cat.DoorName(group[0].DoorID)
	if len(group) > 1 {
		s = ", for " + fmt.Sprint(len(group)) + " of these doors"
	}
	return s
}

// The verdict table, measured off the approved render: verdict at column 2,
// what at 15, door at 49. The door is never shortened — a door printed without
// its real product name is exactly the thing this tool exists to prevent — so
// a long one wraps under its own column and the frame holds.
const (
	verdictW = 13
	whatW    = 34
	doorCol  = 2 + verdictW + whatW
)

func tableRow(verdict, what, door string) []string {
	lines := wrap(door, frameW-doorCol)
	// pad only pads UP. A cell wider than its column ran straight into the next
	// one with no gap — which is exactly what happened when the verdict wording
	// grew from nine characters to eighteen.
	out := []string{"  " + pad(clip(verdict, verdictW-1), verdictW) +
		pad(clip(what, whatW-1), whatW) + lines[0]}
	for _, l := range lines[1:] {
		out = append(out, strings.Repeat(" ", doorCol)+l)
	}
	return out
}

func covWhat(cv credits.Coverage, model string) string {
	if cv.What != "" {
		return cv.What
	}
	return model
}

// withdrawn is the answer that got too old to trust. It reads quiet, because
// the tool going silent should look silent.
func (a app) withdrawn(c credits.Credit, cv credits.Coverage) []string {
	var out []string
	out = append(out, "  "+a.m.of(credits.Declared)+" "+c.Name+" at "+a.cat.DoorName(cv.DoorID))
	out = append(out, "")
	out = append(out, "    "+string(credits.Unknown))
	out = append(out, "")
	out = append(out, "    We used to answer \""+verdictWord(cv.Had)+"\" here. That answer has")
	out = append(out, "    expired, so we took it back.")
	out = append(out, "")
	out = append(out, "      "+pad("what we had", 20)+verdictWord(cv.Had)+", through "+a.cat.DoorName(cv.DoorID))
	out = append(out, "      "+pad("last checked", 20)+fmt.Sprint(cv.AgeDays)+" days ago")
	if cv.Source != "" {
		out = append(out, "      "+pad("source", 20)+strings.TrimPrefix(cv.Source, "https://"))
	}
	out = append(out, "")
	return out
}

func checked(cv credits.Coverage) string {
	when := "checked today"
	if cv.AgeDays == 1 {
		when = "checked 1 day ago"
	} else if cv.AgeDays > 1 {
		when = "checked " + fmt.Sprint(cv.AgeDays) + " days ago"
	}
	if cv.Clock == credits.Stable {
		return when + " against the provider's published terms"
	}
	return when + " by asking the door directly"
}

func what(r credits.Rule) string {
	if r.What != "" {
		return r.What
	}
	if len(r.Models) > 0 {
		return strings.Join(r.Models, ", ")
	}
	return r.Door
}

// The catalog stores "covered" and "not covered". The screen prints what
// happens to the reader.
func verdictWord(v credits.Verdict) string {
	switch v {
	case credits.Covered:
		return "pays for"
	case credits.NotCovered:
		return "not covered"
	default:
		return string(credits.Unknown)
	}
}

const cardW = 38

func (a app) twoDoors(model string, yes, no credits.Coverage) []string {
	f := frame{width: cardW, ascii: a.m.plain}
	// "BILLS YOU" asserted a financial outcome this tool cannot know: whether a
	// call actually charges you depends on your billing account, your project,
	// your free-tier standing and any other credit you hold. What IS knowable is
	// whether the credit covers the door, so that is what the cards say.
	left := a.card(model, yes, "ON YOUR CREDIT")
	right := a.card(model, no, "NOT ON IT")
	for len(left) < len(right) {
		left = append(left, "")
	}
	for len(right) < len(left) {
		right = append(right, "")
	}

	out := []string{"  " + f.top() + "  " + f.heavyTop()}
	for i := range left {
		out = append(out, "  "+f.line(" "+left[i])+"  "+f.heavyLine(" "+right[i]))
	}
	out = append(out, "  "+f.bot()+"  "+f.heavyBot())
	return out
}

func (a app) card(model string, cv credits.Coverage, verdict string) []string {
	inner := cardW - 4
	out := []string{
		clip(model, inner),
		"through   " + clip(a.cat.DoorName(cv.DoorID), inner-10),
	}
	if h := a.cat.DoorHost(cv.DoorID); h != "" {
		out = append(out, clip(h, inner))
	}
	out = append(out, "", " "+verdict, "")
	why := cv.Why
	if why == "" {
		why = "we have no note for this one"
	}
	out = append(out, wrap(why, inner)...)
	return out
}

// ── surface: one credit in full ────────────────────────────────────────────

func (a app) showView(id string) (string, bool) {
	c, ok := a.cat.Credit(id)
	if !ok {
		return "", false
	}
	bal := a.balance(c.ID)
	if bal.Amount != nil {
		c.Balance, c.HasBal = *bal.Amount, true
	}

	var b strings.Builder
	p := func(s string) { b.WriteString(s + "\n") }
	f := a.frame()

	p("")
	p("  " + invocation + " show " + c.ID)
	p("")
	p(f.top())
	p(a.m.row(f, a.m.of(bal.Source), c.Name, hue(c.Colour, a.m.plain),
		leftLabel(c, bal), c.ExpiryLabel(), ageLabel(bal)))
	for _, d := range a.details(c, bal) {
		for _, l := range fitLines(d, f.detailW()) {
			p(f.detail(l))
		}
	}
	p(f.bot())
	p("")

	if c.HowYouGetIt != "" {
		p("  how you get it")
		p("")
		for _, l := range wrap(c.HowYouGetIt, 72) {
			p("    " + l)
		}
		p("")
	}
	if c.Trap != "" {
		p("  worth knowing")
		p("")
		for _, l := range wrap(c.Trap, 72) {
			p("    " + l)
		}
		p("")
	}
	if c.Leftover != "" {
		p("  what happens to the leftover")
		p("")
		p("    " + leftoverWord(c.Leftover))
		p("")
	}
	if c.Source.Where != "" {
		p("  where the balance actually lives")
		p("")
		p("    " + c.Source.Where)
		if c.Source.WhereURL != "" {
			p("    " + strings.TrimPrefix(c.Source.WhereURL, "https://"))
		}
		if c.Source.WhyNotAPI != "" {
			for _, l := range wrap(c.Source.WhyNotAPI, 70) {
				p("    " + l)
			}
		}
		p("")
	}

	p("  " + rule(76))
	p("")
	p("  " + pad("verdict", verdictW) + pad("what", whatW) + "through which door")
	p("")
	for _, want := range []credits.Verdict{credits.Covered, credits.NotCovered, credits.Unknown} {
		n := 0
		for _, r := range c.Rules {
			v, age, _ := r.Assert()
			if v != want {
				continue
			}
			n++
			for _, l := range tableRow(verdictWord(v), what(r), a.cat.DoorName(r.Door)) {
				p(l)
			}
			p("  " + strings.Repeat(" ", verdictW) + ageWord(age) + clockWord(r.Clock))
			if r.StaleRule() {
				p("  " + strings.Repeat(" ", verdictW) + "this may have changed — check the source")
			}
		}
		if n > 0 {
			p("")
		}
	}
	for _, l := range tableRow(string(credits.Unknown), "everything else", "we have not checked it") {
		p(l)
	}
	p("")
	if len(c.Sources) > 0 {
		p("  where we read this")
		p("")
		for _, s := range c.Sources {
			p("    " + strings.TrimPrefix(s, "https://"))
		}
		p("")
	}
	return b.String(), true
}

func ageWord(age int) string {
	switch {
	case age == 0:
		return "checked today · "
	case age == 1:
		return "checked 1 day ago · "
	default:
		return "checked " + fmt.Sprint(age) + " days ago · "
	}
}

func clockWord(ck credits.Clock) string {
	if ck == credits.Stable {
		return "off a published page, good 180 days"
	}
	return "asked the door, quiet after 14 days"
}

func leftoverWord(s string) string {
	switch s {
	case "voids":
		return "whatever you do not spend is lost. Not spending it is not saving it."
	case "carries":
		return "whatever you do not spend stays there."
	case "resets":
		return "it refills on its own schedule. Waste is pure loss, not a saving."
	default:
		return "we do not know."
	}
}
