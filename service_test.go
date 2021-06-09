package musicbrainz

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	md "github.com/gtyrin/go-audiomd"
)

// Тестовые файлы.
const (
	testSearchJSON  = "testdata/search.json"
	testReleaseJSON = "testdata/release.json"
)

func TestSearchResponseParsing(t *testing.T) {
	var out releaseSearchResult
	data, _ := ioutil.ReadFile(testSearchJSON)
	json.Unmarshal(data, &out)
	out.Search()
}

func TestReleaseInfoParsing(t *testing.T) {
	var out releaseInfo
	data, _ := ioutil.ReadFile(testReleaseJSON)
	json.Unmarshal(data, &out)
	release := md.NewRelease()
	out.Release(release)
	release.Optimize()
	// data, _ = json.Marshal(release)
	// ioutil.WriteFile("/home/me/Downloads/test_musicbrainz.json", data, 0755)
	if release.Title != "The Dark Side of the Moon" {
		t.Fail()
	}
}
