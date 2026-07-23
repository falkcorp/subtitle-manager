### Added

#### Provider/score variables and score threshold for custom post-processing

The custom post-processing script now receives `SM_PROVIDER` (the subtitle
provider) and `SM_SCORE` (the quality score, 0-100) in addition to the existing
`SM_SUBTITLE_PATH` / `SM_MEDIA_PATH` / `SM_LANG`, matching Bazarr's
post-processing variables. A new `postprocess.score_threshold` (0-100) gates the
script so it only runs when the subtitle's score meets the threshold; an unset
threshold or an unknown score never gates.
