# Change log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/) 
and this project adheres to [Semantic Versioning](http://semver.org/).

## [1.0.1] - 2017-11-19
### Fixed
- Add canonical import path `cgt.name/pkg/go-mwclient/params` to params
  package.

## [1.0.0] - 2017-01-04
### Added
- OAuth authentication
- Use version 2 of the MediaWiki JSON API by default.
- `New()` sets a 30 second HTTP timeout on the underlying HTTP client.
Can be overridden with the `SetHTTPTimeout()` method.
- New method `AddRange()` on `params.Values` for adding multiple values at once.

### Changed
- `New()` no longer returns an error if the userAgent parameter is empty.
If empty, the default user agent will be used by itself.
- `Login()` and `Logout()` no longer modify the Client's API assertion level.
If assertion is desired, the user must set it manually.
- `Logout()` returns an error if the logout request fails.

### Fixed
- Fix API error parsing in `Login()`.
