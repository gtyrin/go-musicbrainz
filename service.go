package musicbrainz

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"

	md "github.com/ytsiuryn/ds-audiomd"
	srv "github.com/ytsiuryn/ds-service"
)

// Описание сервиса
const (
	ServiceSubsystem   = "audio"
	ServiceName        = "musicbrainz"
	ServiceDescription = "Musicbrainz service client"
)

// Suggestion constants
const (
	MinSearchShortResult = .5
	MinSearchFullResult  = .75
	MaxPreSuggestions    = 7
	MaxSuggestions       = 3
)

// Client constants
const (
	BaseEntityURL      = "https://musicbrainz.org/ws/2/"
	imgURL             = "https://coverartarchive.org"
	releaseParams      = "?inc=annotation+release-groups+artist-credits+recordings+recording-level-rels+artist-rels+genres+labels&fmt=json"
	releaseGroupParams = "?inc=annotation&fmt=json"
	// debugURL = "https://musicbrainz.org/ws/2/release/%s?inc=artist-credits+recordings+recording-level-rels+artist-rels+genres+labels&fmt=json"
	// prodURL        = "https://musicbrainz.org/release/%s"
	// artistDebugURL = "https://musicbrainz.org/ws/2/artist/%s?inc=releases&fmt=json"
	// artistProdURL  = "https://musicbrainz.org/artist/%s"
)

// Известные ошибки.
var (
	ErrAlbumPictureNoAccess = errors.New("invalid character '<' looking for beginning of value")
)

type config struct {
	Auth struct {
		App    string `yaml:"app"`
		Key    string `yaml:"key"`
		Secret string `yaml:"secret"`
		// PersonalToken string `yaml:"personal_token"`
	}
	Product bool `yaml:"product"`
}

// Musicbrainz describes data of Musicbrainz client.
type Musicbrainz struct {
	*srv.PollingService
	conf *config
}

// NewMusicbrainzClient create a new Musicbrainz client.
func NewMusicbrainzClient(connstr, optFile string) (*Musicbrainz, error) {
	conf := config{}
	srv.ReadConfig(optFile, &conf)

	log.SetLevel(srv.LogLevel(conf.Product))

	cl := &Musicbrainz{}
	cl.conf = &conf
	cl.PollingService = srv.NewPollingService(
		map[string]string{
			"User-Agent": cl.conf.Auth.App,
			// "Authorization": "Musicbrainz token=" + cl.conf.Auth.PersonalToken,
		})
	cl.ConnectToMessageBroker(connstr, ServiceName)

	return cl, nil
}

// TestPollingFrequency выполняет определение частоты опроса сервера на примере
// тестового запроса. Периодичность расчитывается в наносекундах.
// TODO: реализовать тестовый запрос.
func (m *Musicbrainz) TestPollingFrequency() error {
	m.SetPollingFrequency(int64(60 * 1000_000_000 / int64(60)))
	return nil
}

// Cleanup ..
func (m *Musicbrainz) Cleanup() {
	m.Service.Cleanup()
}

// RunCmdByName execute a general command and return the result to the client.
func (m *Musicbrainz) RunCmdByName(cmd string, delivery *amqp.Delivery) {
	switch cmd {
	case "search":
		go m.search(delivery)
	case "info":
		version := srv.Version{
			Subsystem:   ServiceSubsystem,
			Name:        ServiceName,
			Description: ServiceDescription,
		}
		go m.Service.Info(delivery, &version)
	default:
		m.Service.RunCommonCmd(cmd, delivery)
	}
}

func (m *Musicbrainz) search(delivery *amqp.Delivery) {
	if m.Idle {
		res := []*md.Suggestion{}
		suggestionsJSON, err := json.Marshal(res)
		if err != nil {
			m.ErrorResult(delivery, err, "Response")
			return
		}
		m.Answer(delivery, suggestionsJSON)
		return
	}
	// прием входного запроса
	var request srv.Request
	err := json.Unmarshal(delivery.Body, &request)
	if err != nil {
		m.ErrorResult(delivery, err, "Request")
		return
	}
	// разбор параметров входного запроса
	var suggestions []*md.Suggestion
	if _, ok := request.Params["release_id"]; ok {
		suggestions, err = m.searchReleaseByID(&request)
		if err != nil {
			m.ErrorResult(delivery, err, "Release by ID")
			return
		}
	} else if _, ok := request.Params["release"]; ok {
		suggestions, err = m.searchReleaseByIncompleteData(&request)
		if err != nil {
			m.ErrorResult(delivery, err, "Release by incomplete data")
			return
		}
	}
	for _, suggestion := range suggestions {
		suggestion.Optimize()
	}
	// отправка ответа
	suggestionsJSON, err := json.Marshal(suggestions)
	if err != nil {
		m.ErrorResult(delivery, err, "Response")
		return
	}
	if !m.conf.Product {
		log.Println(string(suggestionsJSON))
	}
	m.Answer(delivery, suggestionsJSON)
}

func (m *Musicbrainz) searchReleaseByID(request *srv.Request) ([]*md.Suggestion, error) {
	id := request.Params["release_id"]
	r := md.NewRelease()
	if err := m.releaseByID(id, r); err != nil {
		return nil, err
	}
	return []*md.Suggestion{
			{
				Release:          r,
				ServiceName:      ServiceName,
				OnlineSuggeston:  true,
				SourceSimilarity: 1.,
			}},
		nil
}

func (m *Musicbrainz) searchReleaseByIncompleteData(request *srv.Request) ([]*md.Suggestion, error) {
	var suggestions []*md.Suggestion
	// params
	release, err := request.ParseRelease()
	if err != nil {
		return nil, err
	}
	// musicbrainz release search...
	var preResult releaseSearchResult
	if err := m.LoadAndDecode(searchURL(release), &preResult); err != nil {
		return nil, err
	}
	var score float64
	// предварительные предложения
	for _, r := range preResult.Search() {
		if score = release.Compare(r); score > MinSearchShortResult {
			suggestions = append(
				suggestions,
				&md.Suggestion{
					Release:          r,
					ServiceName:      ServiceName,
					OnlineSuggeston:  true,
					SourceSimilarity: score,
				})
		}
	}
	suggestions = md.BestNResults(suggestions, MaxPreSuggestions)
	log.WithField("results", len(suggestions)).Debug("Preliminary search")
	// окончательные предложения
	for i := len(suggestions) - 1; i >= 0; i-- {
		r := suggestions[i].Release
		if err := m.releaseByID(r.IDs[ServiceName], r); err != nil {
			return nil, err
		}
		if score = release.Compare(r); score > MinSearchFullResult {
			suggestions[i].Release = r
			suggestions[i].SourceSimilarity = score
		} else {
			suggestions = append(suggestions[:i], suggestions[i+1:]...)
		}
	}
	suggestions = md.BestNResults(suggestions, MaxSuggestions)
	log.WithField("results", len(suggestions)).Debug("Suggestions")
	return suggestions, nil
}

func (m *Musicbrainz) releaseByID(id string, release *md.Release) error {
	// release request...
	var releaseResp releaseInfo
	if err := m.LoadAndDecode(BaseEntityURL+"release/"+id+releaseParams, &releaseResp); err != nil {
		return err
	}
	releaseResp.Release(release)
	return nil
}

func (m *Musicbrainz) pictures(entityType, id string) ([]*md.PictureInAudio, error) {
	var ret []*md.PictureInAudio
	var ci coverInfo
	err := m.LoadAndDecode(coverURL(entityType, id), &ci)
	if err != nil {
		return nil, err
	}
	if cover := ci.Cover(); cover != nil {
		ret = append(ret, cover)
	}
	return ret, nil
}

func searchURL(release *md.Release) string {
	p := []string{}
	if performers := release.ActorRoles.Filter(md.IsPerformer); len(performers) > 0 {
		firstPerformer := performers.First()
		if firstPerformer != "" {
			if arid, ok := release.Actors[firstPerformer][ServiceName]; ok { // MUSICBRAINZ_ALBUMARTISTID
				p = append(p, queryParam("arid", arid))
			} else {
				p = append(p, queryParam("artist", string(firstPerformer)))
			}
		}
	}
	if release.Title != "" {
		p = append(p, queryParam("release", release.Title))
	}
	if len(release.Publishing) > 0 {
		if len(release.Publishing[0].Name) > 0 {
			p = append(p, queryParam("label", release.Publishing[0].Name))
		}
		if len(release.Publishing[0].Catno) > 0 {
			p = append(p, queryParam("catno", release.Publishing[0].Catno))
		}
	}
	// if release.Year != 0 {
	// 	p = append(p, queryParam("date", strconv.Itoa(int(release.Year))))
	// }
	buffer := bytes.NewBufferString(BaseEntityURL)
	buffer.WriteString("release?query=")
	buffer.WriteString(url.PathEscape(strings.Join(p, " AND ")))
	buffer.WriteString("&fmt=json")
	return buffer.String()
}

func coverURL(entity, releaseID string) string {
	return imgURL + "/" + entity + "/" + releaseID
}

func queryParam(k, v string) string {
	return fmt.Sprintf("%s:\"%s\"", k, v)
}
