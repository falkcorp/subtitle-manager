// file: pkg/arr/filter_test.go
// version: 1.0.0
// guid: d11afe66-6490-452f-be08-c52826b24f04

package arr

import "testing"

func TestTypeExcluded(t *testing.T) {
	f := Filters{ExcludedSeriesTypes: []string{"anime", "daily"}}
	if !f.TypeExcluded("Anime") {
		t.Fatal("expected Anime excluded (case-insensitive)")
	}
	if f.TypeExcluded("standard") {
		t.Fatal("standard should not be excluded")
	}
}

func TestTagsExcluded(t *testing.T) {
	f := Filters{ExcludedTagIDs: []int{5, 8}}
	if !f.TagsExcluded([]int{1, 8}) {
		t.Fatal("expected excluded via tag 8")
	}
	if f.TagsExcluded([]int{1, 2}) {
		t.Fatal("no excluded tag present")
	}
	if (Filters{}).TagsExcluded([]int{1}) {
		t.Fatal("empty exclusion set should never exclude")
	}
}

func TestMapPathLongestPrefix(t *testing.T) {
	f := Filters{PathMappings: [][2]string{
		{"/tv", "/media/tv"},
		{"/tv/anime", "/media/anime"},
	}}
	if got := f.MapPath("/tv/anime/Show/ep.mkv"); got != "/media/anime/Show/ep.mkv" {
		t.Fatalf("longest-prefix mapping failed: %s", got)
	}
	if got := f.MapPath("/tv/Show/ep.mkv"); got != "/media/tv/Show/ep.mkv" {
		t.Fatalf("prefix mapping failed: %s", got)
	}
	if got := f.MapPath("/movies/x.mkv"); got != "/movies/x.mkv" {
		t.Fatalf("unmapped path should pass through: %s", got)
	}
}

func TestIsZero(t *testing.T) {
	if !(Filters{}).IsZero() {
		t.Fatal("empty filters should be zero")
	}
	if (Filters{MonitoredOnly: true}).IsZero() {
		t.Fatal("MonitoredOnly set should not be zero")
	}
}
