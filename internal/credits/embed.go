package credits

import "embed"

//go:embed data/doors.json
var doorsJSON []byte

//go:embed data/credits
var builtinCredits embed.FS
