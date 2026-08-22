// Package agents ships the house's tuned local models as installable things.
//
// The founder's ruling decides the whole shape: "nah let them bring their own."
// Nothing here calls anything he pays for. No proxy, no relay, no shared key,
// no free tier that quietly costs him. The user brings a machine or brings a
// key; what they get from us is a text file and a checked pin.
//
// The reason software wraps twenty lines of Modelfile: the CLI does not run the
// file, it GUARANTEES the artefact — resolves a pin that moved, proves the
// character still behaves, and never silently hands you a different model
// wearing a name it did not earn.
package agents

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// A Pin is one candidate source for an agent's weights, and the digest it was
// verified against. Weights drift: quants get re-uploaded, renamed and pulled.
// The digest is what makes a candidate a substitute rather than a stranger.
type Pin struct {
	Ref    string `json:"ref"`
	Digest string `json:"digest"`
	Note   string `json:"note,omitempty"`
}

// An Agent is a character, not a model. The model is what it runs on.
type Agent struct {
	Name string `json:"name"`
	For  string `json:"for"`
	Size string `json:"size"`
	// Ordered. The first that resolves AND matches its digest wins. A candidate
	// that resolves with the wrong digest is refused, never substituted.
	Pins   []Pin             `json:"pins"`
	Params map[string]string `json:"params"`
	System string            `json:"system"`
	// Claims this agent's system prompt makes, each with the ways they are
	// tested. A behavioural promise nobody checks is worth nothing.
	Claims []Claim `json:"claims"`
	// Named out loud so a passing verify cannot be read as a whole-agent pass.
	Untested []string `json:"untested"`
	// Measured limits — things verify DID test and found, which a green claim
	// would otherwise let a reader conclude away. "Held" and "holds under every
	// way you might call it" are different sentences.
	Known []string `json:"known,omitempty"`
}

// A Claim is something the character promises, and how it is asked.
type Claim struct {
	Says string `json:"says"`
	Asks []Ask  `json:"asks"`
}

// An Ask is one probe and what a holding agent must do with it.
type Ask struct {
	Prompt string `json:"prompt"`
	// The answer must contain one of Wants, must not contain any of Avoids, and
	// must come in under Under words. An Ask carrying none of the three checks
	// nothing, and is reported as checking nothing rather than as a pass.
	Wants  []string `json:"wants,omitempty"`
	Avoids []string `json:"avoids,omitempty"`
	Under  int      `json:"under,omitempty"`
}

// Modelfile renders the agent for `ollama create`, against a chosen pin.
func (a Agent) Modelfile(p Pin) string {
	var b strings.Builder
	fmt.Fprintf(&b, "FROM %s\n", p.Ref)
	for k, v := range a.Params {
		fmt.Fprintf(&b, "PARAMETER %s %s\n", k, v)
	}
	fmt.Fprintf(&b, "SYSTEM \"\"\"%s\"\"\"\n", a.System)
	return b.String()
}

// Digest of a rendered Modelfile, so an install can prove what it built.
func Digest(s string) string {
	h := sha256.Sum256([]byte(s))
	return "sha256:" + hex.EncodeToString(h[:])
}

func Load(b []byte) ([]Agent, error) {
	var as []Agent
	return as, json.Unmarshal(b, &as)
}
