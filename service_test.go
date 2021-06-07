package client

import (
	md "github.com/gtyrin/ds/audio/metadata"
	"encoding/json"
	"io/ioutil"
	"testing"
)

// Тестовые файлы.
const (
	testSearchJSON  = "../../../../../../testdata/audio/online/musicbrainz/search.json"
	testReleaseJSON = "../../../../../../testdata/audio/online/musicbrainz/release.json"
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
