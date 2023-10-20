# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/) and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## Unreleased

### Added

- NEW `--add-labels` command, accepts newline separated list of key=value pairs
- NEW `--add-annotations` command, accepts newline separated list of key=value pairs
- NEW `--add-all` command, accepts newline separated list of commands + arguments; e.g.:

  ```
  add-subscription postgres
  add-label region=us-west-1
  add-annotation foo=bar
  ```

  This enables support for discovery plugins that may modify multiple entity attributes (e.g. subscriptions + labels).

### Fixed

- The `sensu.io/plugins/sensu-entity-manager/config/patch/annotations` and `sensu.io/plugins/sensu-entity-manager/config/patch/subscriptions` annotations are now supported.
  These were documented in the README previously but there were never implemented.

- Implemented a fix for --add-all attribute to enable support for multiple add-label commands
  and multiple add-annotation commands

### Changed

- Q1 '21 handler maintenance:
  - Updated modules (go get -u && go mod tidy)
  - Add pull_request to lint and test GitHub Actions
  - Ran gofmt on source files
  - README updates
  - Fix linter error for dead code

## [0.0.1] - 2000-01-01

### Added

- Initial release
