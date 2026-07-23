### Added

#### Library paths API for the Media Library

Added the `/api/library/paths` (GET/POST/DELETE) plus `/api/library/rescan` and
`/api/library/resync` endpoints. The Media Library UI called these but no backend
route existed, so the library always appeared empty and "Add Library Path"
silently failed. Configured root paths are now persisted (a JSON file beside the
database) and adding or rescanning a path scans it into the media library.
