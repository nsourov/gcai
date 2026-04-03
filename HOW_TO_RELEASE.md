# How To Release `gcai`

This guide explains how to publish a new `gcai` release with binaries so users can install via:

```bash
curl -fsSL https://raw.githubusercontent.com/nsourov/gcai/main/scripts/install.sh | bash
```

The installer expects release assets named like:

- `gcai_<version_without_v>_<os>_<arch>.tar.gz`
- Example: `gcai_0.1.0_darwin_arm64.tar.gz`

## 1) Choose a version

Use semantic versions with a `v` tag, for example:

- `v0.1.0`
- `v0.1.1`

In commands below:

- `TAG=v0.1.0`
- `VER=0.1.0`

## 2) Build binaries (AMD64 + ARM64)

From the repository root:

```bash
mkdir -p dist
TAG=v0.1.0
VER="${TAG#v}"
```

Always pass `-ldflags "-X main.version=${TAG}"` so `gcai --version` matches the release tag.

### macOS ARM64

```bash
GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=${TAG}" -o dist/gcai ./cmd/gcai
tar -C dist -czf "dist/gcai_${VER}_darwin_arm64.tar.gz" gcai
```

### macOS AMD64

```bash
GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=${TAG}" -o dist/gcai ./cmd/gcai
tar -C dist -czf "dist/gcai_${VER}_darwin_amd64.tar.gz" gcai
```

### Linux ARM64

```bash
GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=${TAG}" -o dist/gcai ./cmd/gcai
tar -C dist -czf "dist/gcai_${VER}_linux_arm64.tar.gz" gcai
```

### Linux AMD64

```bash
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=${TAG}" -o dist/gcai ./cmd/gcai
tar -C dist -czf "dist/gcai_${VER}_linux_amd64.tar.gz" gcai
```

## 3) Create and push git tag

```bash
git tag "$TAG"
git push origin "$TAG"
```

## 4) Create GitHub release and upload assets

Using GitHub CLI:

```bash
gh release create "$TAG" \
  "dist/gcai_${VER}_darwin_arm64.tar.gz" \
  "dist/gcai_${VER}_darwin_amd64.tar.gz" \
  "dist/gcai_${VER}_linux_arm64.tar.gz" \
  "dist/gcai_${VER}_linux_amd64.tar.gz" \
  --title "$TAG" \
  --notes "Release $TAG"
```

Or use GitHub web UI:

1. Open `https://github.com/nsourov/gcai/releases`
2. Draft a new release with tag `vX.Y.Z`
3. Upload all generated `.tar.gz` assets from `dist/`
4. Publish release

## 5) Verify installer

After release is published:

```bash
curl -fsSL https://raw.githubusercontent.com/nsourov/gcai/main/scripts/install.sh | bash
gcai --help
```

## Troubleshooting

- **`Could not resolve release version`**
  - No published release exists yet. Create one first.
- **`requested URL returned error: 404` during asset download**
  - Asset filename does not match expected pattern.
  - Check OS/arch naming exactly (`darwin/linux`, `amd64/arm64`).
- **`Archive did not contain gcai`**
  - Tarball must contain a file named exactly `gcai` at archive root.
