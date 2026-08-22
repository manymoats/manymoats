package version

// V is stamped at build time with -ldflags "-X .../internal/version.V=v0.1.0".
// It stays "dev" for a plain `go build`, which is the honest answer for a binary
// that did not come from a release.
var V = "dev"
