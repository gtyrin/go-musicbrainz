package musicbrainz

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	md "github.com/ytsiuryn/ds-audiomd"
	srv "github.com/ytsiuryn/ds-microservice"
)

// Тестовые файлы.
const (
	testSearchJSON  = "testdata/search.json"
	testReleaseJSON = "testdata/release.json"
)

type MusicbrainzTestSuite struct {
	suite.Suite
	cl *srv.RPCClient
}

func (suite *MusicbrainzTestSuite) SetupSuite() {
	suite.startTestService()
	suite.cl = srv.NewRPCClient()
}

func (suite *MusicbrainzTestSuite) TearDownSuite() {
	suite.cl.Close()
}
func (suite *MusicbrainzTestSuite) TestBaseServiceCommands() {
	correlationID, data, err := srv.CreateCmdRequest("ping")
	require.NoError(suite.T(), err)
	suite.cl.Request(ServiceName, correlationID, data)
	respData := suite.cl.Result(correlationID)
	suite.Empty(respData)

	correlationID, data, err = srv.CreateCmdRequest("x")
	require.NoError(suite.T(), err)
	suite.cl.Request(ServiceName, correlationID, data)
	vInfo, err := srv.ParseErrorAnswer(suite.cl.Result(correlationID))
	require.NoError(suite.T(), err)
	// {"error": "Unknown command: x", "context": "Message dispatcher"}
	suite.Equal(vInfo.Error, "Unknown command: x")
}

func (suite *MusicbrainzTestSuite) TestSearchRelease() {
	r := md.NewRelease()
	r.Title = "The Dark Side Of The Moon"
	r.Year = 1977
	r.ActorRoles.Add("Pink Floyd", "performer")
	r.Publishing = append(r.Publishing, &md.Publishing{Name: "Harvest", Catno: "SHVL 804"})

	correlationID, data, err := CreateReleaseRequest(r)
	require.NoError(suite.T(), err)
	suite.cl.Request(ServiceName, correlationID, data)

	resp, err := ParseReleaseAnswer(suite.cl.Result(correlationID))
	require.NoError(suite.T(), err)
	require.Empty(suite.T(), resp.Error)

	suite.NotEmpty(resp)
	suite.Equal(
		strings.ToLower(resp.SuggestionSet.Suggestions[0].Release.Title),
		"the dark side of the moon")
}

func (suite *MusicbrainzTestSuite) startTestService() {
	testService := New(
		os.Getenv("MUSICBRAINZ_APP"),
		os.Getenv("MUSICBRAINZ_KEY"),
		os.Getenv("MUSICBRAINZ_SECRET"))
	msgs := testService.ConnectToMessageBroker("amqp://guest:guest@localhost:5672/")
	testService.Log.SetLevel(log.DebugLevel)
	go testService.Start(msgs)
}

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

func TestMusicbrainzOnline(t *testing.T) {
	suite.Run(t, new(MusicbrainzTestSuite))
}
