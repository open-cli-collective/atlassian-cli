# Homebrew Distribution

Homebrew packaging is managed automatically by [GoReleaser](https://goreleaser.com/) during the release process.

## Configuration

The Homebrew cask configuration lives in [`.goreleaser.yml`](../../.goreleaser.yml) under the `homebrew_casks` section.

## Tap Repository

The generated cask is published to: https://github.com/open-cli-collective/homebrew-tap

## Installation

```bash
brew install open-cli-collective/tap/cfl
```
