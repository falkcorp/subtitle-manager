### Fixed

#### Failed searches now say why

`FetchFromAll` computed the last provider error, used it for the failure event,
and then discarded it — returning a bare `no subtitle found` for every cause. A
dead host, a malformed response and "this provider has no credentials
configured" were indistinguishable, and the last of those is the only one an
operator can act on.

The underlying error is now wrapped, keeping the familiar prefix so anything
matching on it still works:

    no subtitle found: opensubtitles: authentication failed: ...

Found while checking why a real `fetch` produced nothing: OpenSubtitles was
entirely unconfigured, and the tool had no way of saying so.
