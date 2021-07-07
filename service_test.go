package musicbrainz

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	md "github.com/ytsiuryn/ds-audiomd"
)

// Тестовые файлы.
const (
	testSearchJSON  = "testdata/search.json"
	testReleaseJSON = "testdata/release.json"
)

var mut sync.Mutex
var testService *Musicbrainz

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

func startTestService(ctx context.Context) {
	mut.Lock()
	defer mut.Unlock()
	if testService == nil {
		testService = NewMusicbrainzClient(
			os.Getenv("MUSICBRAINZ_APP"),
			os.Getenv("MUSICBRAINZ_KEY"),
			os.Getenv("MUSICBRAINZ_SECRET"))
		msgs := testService.ConnectToMessageBroker("amqp://guest:guest@localhost:5672/")
		// defer test.Cleanup()
		go testService.Start(msgs)
	}
}
