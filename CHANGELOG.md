# Changelog

## [0.8.1](https://github.com/open-cli-collective/confluence-cli/compare/v0.8.0...v0.8.1) (2026-01-16)


### Bug Fixes

* allow --parent flag to move page without content ([#64](https://github.com/open-cli-collective/confluence-cli/issues/64)) ([863c29d](https://github.com/open-cli-collective/confluence-cli/commit/863c29d869b24a8a194fccc2f4401eedcdbc35ad)), closes [#60](https://github.com/open-cli-collective/confluence-cli/issues/60)
* correct homebrew tap reference ([#63](https://github.com/open-cli-collective/confluence-cli/issues/63)) ([949c85e](https://github.com/open-cli-collective/confluence-cli/commit/949c85e982c34bd1b89565c0def544121c0bda58))
* validate empty content client-side before API call ([#61](https://github.com/open-cli-collective/confluence-cli/issues/61)) ([f122c95](https://github.com/open-cli-collective/confluence-cli/commit/f122c95c928b57458362f3547a1ca94cdcbe92bb)), closes [#59](https://github.com/open-cli-collective/confluence-cli/issues/59)

## [0.8.0](https://github.com/rianjs/confluence-cli/compare/v0.7.0...v0.8.0) (2026-01-15)


### Features

* Support common Confluence macros (TOC, panels, expand) ([#52](https://github.com/rianjs/confluence-cli/issues/52)) ([e991f9f](https://github.com/rianjs/confluence-cli/commit/e991f9f873212bf7c11bc60674df0c51dfa7f6c0))

## [0.7.0](https://github.com/rianjs/confluence-cli/compare/v0.6.0...v0.7.0) (2026-01-14)


### Features

* add --parent flag to page edit command ([#48](https://github.com/rianjs/confluence-cli/issues/48)) ([a66160d](https://github.com/rianjs/confluence-cli/commit/a66160d4c7d38e333f33d8a1e89cc5692b43c046))

## [0.6.0](https://github.com/rianjs/confluence-cli/compare/v0.5.0...v0.6.0) (2026-01-14)


### Features

* add shell completion support ([#44](https://github.com/rianjs/confluence-cli/issues/44)) ([10dbc24](https://github.com/rianjs/confluence-cli/commit/10dbc245b8f09c347c567ca29687b074dbf80ec0)), closes [#43](https://github.com/rianjs/confluence-cli/issues/43)

## [0.5.0](https://github.com/rianjs/confluence-cli/compare/v0.4.0...v0.5.0) (2026-01-13)


### Features

* add cloud editor (ADF) support for page creation ([#40](https://github.com/rianjs/confluence-cli/issues/40)) ([ae8eb8b](https://github.com/rianjs/confluence-cli/commit/ae8eb8b1aa5654ea7cf1085b4bd5229698f75ebf))

## [0.4.0](https://github.com/rianjs/confluence-cli/compare/v0.3.2...v0.4.0) (2026-01-12)


### Features

* add Confluence search with CQL support ([#37](https://github.com/rianjs/confluence-cli/issues/37)) ([bda490c](https://github.com/rianjs/confluence-cli/commit/bda490c9287698fa71ca13a0f6d5789607557526)), closes [#36](https://github.com/rianjs/confluence-cli/issues/36)

## [0.3.2](https://github.com/rianjs/confluence-cli/compare/v0.3.1...v0.3.2) (2026-01-12)


### Bug Fixes

* enable GFM table extension in markdown converter ([#34](https://github.com/rianjs/confluence-cli/issues/34)) ([a21ca4f](https://github.com/rianjs/confluence-cli/commit/a21ca4f15a3c6ccc1d9de89c8ccc1c00a49027a8)), closes [#30](https://github.com/rianjs/confluence-cli/issues/30)

## [0.3.1](https://github.com/rianjs/confluence-cli/compare/v0.3.0...v0.3.1) (2026-01-12)


### Bug Fixes

* add _meta field to JSON list output for pagination ([#32](https://github.com/rianjs/confluence-cli/issues/32)) ([0005918](https://github.com/rianjs/confluence-cli/commit/0005918a8b59abaec41099a1c410017d6a78849a))

## [0.3.0](https://github.com/rianjs/confluence-cli/compare/v0.2.5...v0.3.0) (2026-01-11)


### Features

* add --unused flag to find orphaned attachments ([#28](https://github.com/rianjs/confluence-cli/issues/28)) ([c7653e5](https://github.com/rianjs/confluence-cli/commit/c7653e510951226a851bb4ff4944b50aec814413))

## [0.2.5](https://github.com/rianjs/confluence-cli/compare/v0.2.4...v0.2.5) (2026-01-11)


### Bug Fixes

* preserve tables in HTML to markdown conversion ([#26](https://github.com/rianjs/confluence-cli/issues/26)) ([56340da](https://github.com/rianjs/confluence-cli/commit/56340da499dd0e73cdba6cd4b71ba32e5860d989))

## [0.2.4](https://github.com/rianjs/confluence-cli/compare/v0.2.3...v0.2.4) (2026-01-11)


### Bug Fixes

* preserve code blocks from Confluence UI pages in markdown output ([#24](https://github.com/rianjs/confluence-cli/issues/24)) ([b29653b](https://github.com/rianjs/confluence-cli/commit/b29653bfb37f201181cdb01ca12044719bdfc5f0)), closes [#15](https://github.com/rianjs/confluence-cli/issues/15)

## [0.2.3](https://github.com/rianjs/confluence-cli/compare/v0.2.2...v0.2.3) (2026-01-11)


### Bug Fixes

* reject invalid --status values with helpful error message ([#22](https://github.com/rianjs/confluence-cli/issues/22)) ([ee58d7f](https://github.com/rianjs/confluence-cli/commit/ee58d7f9a8b708ae9a63c48261e2604c540cefd9))

## [0.2.2](https://github.com/rianjs/confluence-cli/compare/v0.2.1...v0.2.2) (2026-01-11)


### Bug Fixes

* resolve space key from spaceId for page copy ([4b85f2a](https://github.com/rianjs/confluence-cli/commit/4b85f2ab51d9ae7486905e12c701943b5c5c92c1))

## [0.2.1](https://github.com/rianjs/confluence-cli/compare/v0.2.0...v0.2.1) (2026-01-11)


### Bug Fixes

* use downloadLink from attachment metadata for downloads ([1621fd8](https://github.com/rianjs/confluence-cli/commit/1621fd885ae451dc2ea8c80f533a3b9c1cd62ee4))
* use downloadLink from attachment metadata for downloads ([42e3053](https://github.com/rianjs/confluence-cli/commit/42e3053e942ff7e131f6f4aade4bab4906e56d42))
* use RELEASE_PAT to trigger release workflow ([3a9fd3c](https://github.com/rianjs/confluence-cli/commit/3a9fd3c5791bb2d21a4c75d747f7d8c4d39172ab))

## [0.2.0](https://github.com/rianjs/confluence-cli/compare/v0.1.1...v0.2.0) (2026-01-10)


### Features

* add attachment delete command ([a9f3d25](https://github.com/rianjs/confluence-cli/commit/a9f3d259281ece084a574855ab1fa9a37d102443))
* add automated releases via release-please ([91d7354](https://github.com/rianjs/confluence-cli/commit/91d7354b59beb7648fe3a30aafc089c00f75622d))
* add automated releases via release-please ([f2d6212](https://github.com/rianjs/confluence-cli/commit/f2d62122120e3c3d282a1ed1d89f39d39c0f1dc6))
* add page copy command ([f86c9ee](https://github.com/rianjs/confluence-cli/commit/f86c9ee72e5b890f2f35ca96854b4cca19e4b43e))
* add page copy command ([7290a3e](https://github.com/rianjs/confluence-cli/commit/7290a3e431cef15da84bfe5b73764dab58c99936))
* warn before overwriting existing files in attachment download ([32a5445](https://github.com/rianjs/confluence-cli/commit/32a5445b8da4691ce77f14beca214dfb9f126cb9))


### Bug Fixes

* pin golangci-lint to v2 in CI ([6bb768e](https://github.com/rianjs/confluence-cli/commit/6bb768e439d9c3250521f4ca8809613fa4f861ed))
* resolve 6 bugs found during chaos testing ([ddc6f77](https://github.com/rianjs/confluence-cli/commit/ddc6f7749c231080dd0c2285829c42bec009c58b))
* resolve golangci-lint v2 issues ([fd699f7](https://github.com/rianjs/confluence-cli/commit/fd699f7617ff2ec2b0174452135211db30dc87d0))
* resolve golangci-lint v2 issues ([2762523](https://github.com/rianjs/confluence-cli/commit/2762523e348626e6cafbfc98f9b2692a50c903d1))
* sanitize attachment download filenames to prevent path traversal ([8f35b16](https://github.com/rianjs/confluence-cli/commit/8f35b16e3f85b02ae16dc467102624b316ba33fd))
