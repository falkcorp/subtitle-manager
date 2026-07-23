### Fixed

#### Data race in async library-path scan

`scanPathAsync` opened the database store (which reads viper config) inside its
background goroutine, which could race with a concurrent viper mutation — e.g.
a test's deferred `viper.Reset()` — and was caught by the `-race` detector in
`TestLibraryPathsHandler`. The store is now opened synchronously before the
goroutine starts; only the scan itself runs in the background.
