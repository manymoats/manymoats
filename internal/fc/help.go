package fc

import "strings"

// The maker's mark exists here and in --version. A normal run carries nothing:
// no banner, no nudge, no phone-home.
const makerLine = "no telemetry. we collect nothing. made by manymoats · manymoats.com"

const helpBody = `manymoats credits — what your free AI credits actually pay for

  manymoats credits
      your credits: what's left, when each one dies
  manymoats credits covers <model>
      which of your credits pay for this model, and through which door
  manymoats credits show <credit>
      one credit in full: what it pays for, what it doesn't, what we
      don't know
  manymoats credits list
      every credit we know of, held by you or not

  not built yet, though the plan has them: route, check, watch, record,
  trust, update, validate. They are named here so you know what is missing.

    --json                    machine-readable output
    --no-network              answer from built-in data only; contact nobody
    --yes                     ask nothing; safe inside scripts and CI
    --plain                   no colour, no symbols
    --snapshot                print one frame and exit, for checking output
    --holdings <file>         read your own figures from this file

  A door is the web address you send the request to. The same model often has
  two doors, and they don't bill the same. That's the whole reason this exists.

`

const helpTail = `
  We never show a number we can't check as if it were current.
  When we don't know, we print "unknown".
  An answer about what a plan includes goes quiet after 14 days; an answer
  read off a published terms page lasts 180. Both print their age.
  Promotional terms change without notice. Check the provider's own terms
  before you rely on anything here.

`

func helpText(m marks) string {
	var b strings.Builder
	b.WriteString(helpBody)
	if m.plain {
		b.WriteString("  live      we asked the provider just now\n")
		b.WriteString("  part      a known starting figure minus spending we measured\n")
		b.WriteString("  old       written down by hand — we always print how old it is\n")
	} else {
		b.WriteString("  ● live      we asked the provider just now\n")
		b.WriteString("  ◐ derived   a known starting figure minus spending we measured\n")
		b.WriteString("  ○ declared  written down by hand — we always print how old it is\n")
	}
	b.WriteString(helpTail)
	b.WriteString("  " + makerLine + "\n")
	return b.String()
}
