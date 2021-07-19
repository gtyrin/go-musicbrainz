package musicbrainz

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	md "github.com/ytsiuryn/ds-audiomd"
	srv "github.com/ytsiuryn/ds-microservice"
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

func TestOfflineParsing(t *testing.T) {
	var out releaseInfo
	data, _ := ioutil.ReadFile(testReleaseJSON)
	json.Unmarshal(data, &out)
	release := md.NewRelease()
	out.Release(release)
	release.Optimize()
	assert.Equal(t, release.Title, "The Dark Side of the Moon")
}

func TestBaseServiceCommands(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startTestService(ctx)

	cl := srv.NewRPCClient()
	defer cl.Close()

	correlationID, data, err := srv.CreateCmdRequest("ping")
	require.NoError(t, err)
	cl.Request(ServiceName, correlationID, data)
	respData := cl.Result(correlationID)
	assert.Len(t, respData, 0)

	correlationID, data, err = srv.CreateCmdRequest("x")
	require.NoError(t, err)
	cl.Request(ServiceName, correlationID, data)
	vInfo, err := srv.ParseErrorAnswer(cl.Result(correlationID))
	require.NoError(t, err)
	// {"error": "Unknown command: x", "context": "Message dispatcher"}
	assert.Equal(t, vInfo.Error, "Unknown command: x")
}

func TestSearchRelease(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startTestService(ctx)

	cl := srv.NewRPCClient()
	defer cl.Close()

	r := md.NewRelease()
	r.Title = "The Dark Side Of The Moon"
	r.Year = 1977
	r.ActorRoles.Add("Pink Floyd", "performer")
	r.Publishing = append(r.Publishing, &md.Publishing{Name: "Harvest", Catno: "SHVL 804"})

	correlationID, data, err := CreateReleaseRequest(r)
	require.NoError(t, err)
	cl.Request(ServiceName, correlationID, data)

	suggestions, err := ParseReleaseAnswer(cl.Result(correlationID))
	require.NoError(t, err)

	assert.NotEmpty(t, suggestions)
	assert.Equal(t, strings.ToLower(suggestions[0].Release.Title), "the dark side of the moon")
}

func startTestService(ctx context.Context) {
	mut.Lock()
	defer mut.Unlock()
	if testService == nil {
		testService = New(
			os.Getenv("MUSICBRAINZ_APP"),
			os.Getenv("MUSICBRAINZ_KEY"),
			os.Getenv("MUSICBRAINZ_SECRET"))
		msgs := testService.ConnectToMessageBroker("amqp://guest:guest@localhost:5672/")
		testService.Log.SetLevel(log.DebugLevel)
		go testService.Start(msgs)
	}
}
