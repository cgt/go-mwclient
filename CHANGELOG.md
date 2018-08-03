# Change log

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/) 
and this project adheres to [Semantic Versioning](http://semver.org/).

Changes that are required to maintain compatibility with new versions of
MediaWiki are not considered breaking changes.

## [1.0.3] - 2018-08-03
### Fixed
- *Get page* functions no longer treat warnings as fatal errors. Return pages
  along with warnings instead of only returning the warnings if there are any.
  Fixes issue #9.

## [1.0.2] - 2018-08-02
### Fixed
- Add `"rvslots": "main"` to the *get page* (`prop=revisions`) requests.
  This change affects `GetPageByID`, `GetPageByName`, `GetPagesByID`,
  and `GetPagesByName`.  The `rvslots` parameter is required by MediaWiki
  [1.32.0-wmf.15](https://gerrit.wikimedia.org/r/plugins/gitiles/mediawiki/core/+/07842be379ca3d4d0bc0608c217dd0e8cd7cbe4b),
  which returns a warning if it is not used. Earlier versions
  of MediaWiki will return an "Unrecognized parameter: rvslots." warning when
  the parameter is used.

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
