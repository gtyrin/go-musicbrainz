package musicbrainz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/streadway/amqp"

	md "github.com/ytsiuryn/ds-audiomd"
	srv "github.com/ytsiuryn/ds-microservice"
)

// Описание сервиса
const ServiceName = "musicbrainz"

// Suggestion constants
const (
	MinSearchShortResult = .5
	MinSearchFullResult  = .75
	MaxPreSuggestions    = 7
	MaxSuggestions       = 3
)

// Client constants
const (
	BaseURL       = "https://musicbrainz.org/ws/2/"
	ImgURL        = "https://coverartarchive.org"
	releaseParams = "?inc=annotation+release-groups+artist-credits+recordings+recording-level-rels+artist-rels+genres+labels&fmt=json"
	// releaseGroupParams = "?inc=annotation&fmt=json"
	// debugURL = "https://musicbrainz.org/ws/2/release/%s?inc=artist-credits+recordings+recording-level-rels+artist-rels+genres+labels&fmt=json"
	// prodURL        = "https://musicbrainz.org/release/%s"
	// artistDebugURL = "https://musicbrainz.org/ws/2/artist/%s?inc=releases&fmt=json"
	// artistProdURL  = "https://musicbrainz.org/artist/%s"
)

// Musicbrainz describes data of Musicbrainz client.
type Musicbrainz struct {
	*srv.Service
	headers map[string]string
	poller  *srv.WebPoller
}

// New create a new Musicbrainz client.
func New(app, key, secret string) *Musicbrainz {
	ret := &Musicbrainz{
		Service: srv.NewService(ServiceName),
		headers: map[string]string{
			"User-Agent": app,
			// "Authorization": "Musicbrainz token=" + cl.conf.Auth.PersonalToken,
		},
		poller: srv.NewWebPoller(2500 * time.Millisecond)}
	ret.poller.Log = ret.Log
	return ret
}

// TestPollingInterval выполняет определение частоты опроса сервера на примере
// тестового запроса. Периодичность расчитывается в наносекундах.
// TODO: реализовать тестовый запрос.
func (m *Musicbrainz) TestPollingInterval() {
	// m.Log.Info("Polling interval: ", m.poller.PollingInterval())
}

// Start запускает Web Poller и цикл обработки взодящих запросов.
// Контролирует сигнал завершения цикла и последующего освобождения ресурсов микросервиса.
func (m *Musicbrainz) Start(msgs <-chan amqp.Delivery) {
	m.poller.Start()
	go m.TestPollingInterval()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		for delivery := range msgs {
			var req AudioOnlineRequest
			if err := json.Unmarshal(delivery.Body, &req); err != nil {
				m.AnswerWithError(&delivery, err, "Message dispatcher")
				continue
			}
			m.logRequest(&req)
			m.RunCmd(&req, &delivery)
		}
	}()

	m.Log.Info("Awaiting RPC requests")
	<-c

	m.cleanup()
}

func (m *Musicbrainz) cleanup() {
	m.Service.Cleanup()
}

// Отображение сведений о выполняемом запросе.
func (m *Musicbrainz) logRequest(req *AudioOnlineRequest) {
	if req.Release != nil {
		if _, ok := req.Release.IDs[ServiceName]; ok {
			m.Log.WithField("args", req.Release.IDs[ServiceName]).Info(req.Cmd + "()")
		} else { // TODO: может стоит офомить метод String() для md.Release?
			var args []string
			if actor := string(req.Release.ActorRoles.Filter(md.IsPerformer).First()); actor != "" {
				args = append(args, actor)
			}
			if req.Release.Title != "" {
				args = append(args, req.Release.Title)
			}
			if req.Release.Year != 0 {
				args = append(args, strconv.Itoa(req.Release.Year))
			}
			m.Log.WithField("args", strings.Join(args, "-")).Info(req.Cmd + "()")
		}
	} else {
		m.Log.Info(req.Cmd + "()")
	}
}

// RunCmd вызывает командам  запроса методы сервиса и возвращает результат клиенту.
func (m *Musicbrainz) RunCmd(req *AudioOnlineRequest, delivery *amqp.Delivery) {
	switch req.Cmd {
	case "release":
		go m.release(req, delivery)
	default:
		m.Service.RunCmd(req.Cmd, delivery)
	}
}

func (m *Musicbrainz) release(request *AudioOnlineRequest, delivery *amqp.Delivery) {
	// разбор параметров входного запроса
	var err error
	var suggestions []*md.Suggestion
	if _, ok := request.Release.IDs[ServiceName]; ok {
		suggestions, err = m.searchReleaseByID(request.Release.IDs[ServiceName])
	} else {
		suggestions, err = m.searchReleaseByIncompleteData(request.Release)
	}
	if err != nil {
		m.AnswerWithError(delivery, err, "Getting release data")
		return
	}
	for _, suggestion := range suggestions {
		suggestion.Optimize()
	}
	// отправка ответа
	suggestionsJSON, err := json.Marshal(suggestions)
	if err != nil {
		m.AnswerWithError(delivery, err, "Response")
	} else {
		m.Log.Debug(string(suggestionsJSON))
		m.Answer(delivery, suggestionsJSON)
	}
}

func (m *Musicbrainz) searchReleaseByID(id string) ([]*md.Suggestion, error) {
	r := md.NewRelease()
	if err := m.releaseByID(id, r); err != nil {
		return nil, err
	}
	return []*md.Suggestion{
			{
				Release:          r,
				ServiceName:      ServiceName,
				SourceSimilarity: 1.,
			}},
		nil
}

func (m *Musicbrainz) searchReleaseByIncompleteData(release *md.Release) ([]*md.Suggestion, error) {
	var suggestions []*md.Suggestion
	// musicbrainz release search...
	var preResult releaseSearchResult
	if err := m.poller.Decode(searchURL(release), m.headers, &preResult); err != nil {
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
					SourceSimilarity: score,
				})
		}
	}
	suggestions = md.BestNResults(suggestions, MaxPreSuggestions)
	m.Log.WithField("results", len(suggestions)).Debug("Preliminary search")
	// окончательные предложения
	for i := len(suggestions) - 1; i >= 0; i-- {
		r := suggestions[i].Release
		if err := m.releaseByID(r.IDs[ServiceName], r); err != nil {
			return nil, err
		}
		if score = release.Compare(r); score > MinSearchFullResult {
			suggestions[i].SourceSimilarity = score
		} else {
			suggestions = append(suggestions[:i], suggestions[i+1:]...)
		}
	}
	suggestions = md.BestNResults(suggestions, MaxSuggestions)
	m.Log.WithField("results", len(suggestions)).Debug("Suggestions")
	return suggestions, nil
}

func (m *Musicbrainz) releaseByID(id string, release *md.Release) error {
	// release request...
	var releaseResp releaseInfo
	if err := m.poller.Decode(
		BaseURL+"release/"+id+releaseParams, m.headers, &releaseResp); err != nil {
		return err
	}
	releaseResp.Release(release)
	return nil
}

func (m *Musicbrainz) pictures(entityType, id string) ([]*md.PictureInAudio, error) {
	var ret []*md.PictureInAudio
	var ci coverInfo
	if err := m.poller.Decode(coverURL(entityType, id), m.headers, &ci); err != nil {
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
	buffer := bytes.NewBufferString(BaseURL)
	buffer.WriteString("release?query=")
	buffer.WriteString(url.PathEscape(strings.Join(p, " AND ")))
	buffer.WriteString("&fmt=json")
	return buffer.String()
}

func coverURL(entity, releaseID string) string {
	return ImgURL + "/" + entity + "/" + releaseID
}

func queryParam(k, v string) string {
	return fmt.Sprintf("%s:\"%s\"", k, v)
}
