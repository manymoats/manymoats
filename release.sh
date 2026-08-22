#!/usr/bin/env bash
# Ship a version. ./release.sh v0.1.1
#
# Builds both Mac architectures, publishes a GitHub release, and points the
# Homebrew formula at it. After this runs, `brew upgrade manymoats` gets it.
set -euo pipefail

V="${1:-}"
[[ "$V" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "usage: ./release.sh v0.1.1"; exit 1; }
REPO=manymoats/manymoats
TAP=manymoats/homebrew-tap
BARE="${V#v}"

[[ -z "$(git status --porcelain)" ]] || { echo "working tree is dirty — commit first"; exit 1; }
go vet ./... && go test ./...

rm -rf dist && mkdir -p dist
for ARCH in arm64 amd64; do
  echo "  building darwin/$ARCH"
  GOOS=darwin GOARCH=$ARCH CGO_ENABLED=0 go build \
    -ldflags "-s -w -X github.com/manymoats/manymoats/internal/version.V=$V" \
    -o "dist/manymoats" .
  codesign -f -s - "dist/manymoats"
  tar -czf "dist/manymoats_${BARE}_darwin_${ARCH}.tar.gz" -C dist manymoats
  rm dist/manymoats
done
( cd dist && shasum -a 256 *.tar.gz > checksums.txt && cat checksums.txt )

git tag -a "$V" -m "$V" && git push origin main --tags
gh release create "$V" dist/*.tar.gz dist/checksums.txt --repo "$REPO" --title "$V" --generate-notes

SHA_ARM=$(awk '/arm64/{print $1}' dist/checksums.txt)
SHA_X86=$(awk '/amd64/{print $1}' dist/checksums.txt)

T=$(mktemp -d) && git clone -q "https://github.com/$TAP.git" "$T"
mkdir -p "$T/Formula"
cat > "$T/Formula/manymoats.rb" <<RB
class Manymoats < Formula
  desc "Terminal tools from manymoats — a status bar for your coding agents"
  homepage "https://github.com/manymoats/manymoats"
  version "$BARE"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/manymoats/manymoats/releases/download/$V/manymoats_${BARE}_darwin_arm64.tar.gz"
      sha256 "$SHA_ARM"
    end
    on_intel do
      url "https://github.com/manymoats/manymoats/releases/download/$V/manymoats_${BARE}_darwin_amd64.tar.gz"
      sha256 "$SHA_X86"
    end
  end

  def install
    bin.install "manymoats"
  end

  test do
    assert_match "manymoats", shell_output("#{bin}/manymoats --version")
  end
end
RB
git -C "$T" add -A && git -C "$T" commit -qm "manymoats $V" && git -C "$T" push -q
rm -rf "$T"
echo
echo "  shipped $V — brew upgrade manymoats"
