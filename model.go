package musicbrainz

import (
	"strconv"
	"strings"

	md "github.com/ytsiuryn/ds-audiomd"
	collection "github.com/ytsiuryn/go-collection"
	intutils "github.com/ytsiuryn/go-intutils"
	tp "github.com/ytsiuryn/go-stringutils"
)

// coverarchive.org structure:

type thumbnail struct {
	URLLarge string `json:"large"`
	URLSmall string `json:"small"`
}

type imageInfo struct {
	Edit int32 `json:"edit"`
	// id         string    `json:"id"`
	ImageURL   string    `json:"image"`
	Thumbnails thumbnail `json:"thumbnails"`
	Comment    string    `json:"comment"`
	Approved   bool      `json:"approved"`
	Front      bool      `json:"front"`
	Types      []string  `json:"types"`
	Back       bool      `json:"back"`
}

// type artist struct {
// 	ID       string `json:"id"`
// 	Name     string `json:"name"`
// 	SortName string `json:"sort-name"`
// }

type label struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type labelInfo struct {
	CatalogNumber string `json:"catalog-number"`
	Label         label  `json:"label"`
}

type media struct {
	Format     string `json:"format"`
	DiscCount  int32  `json:"disc-count"`
	TrackCount int32  `json:"track-count"`
}

// type coverArtArchive struct {
// 	Artwork  bool  `json:"artwork"`
// 	Front    bool  `json:"front"`
// 	Back     bool  `json:"back"`
// 	Count    int32 `json:"count"`
// 	Darkened bool  `json:"darkened"`
// }

type textRepresentation struct {
	Script   string `json:"script"`
	Language string `json:"language"`
}

type trackArtist struct {
	// Disambiguation string `json:"disambiguation"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	SortName string `json:"sort-name"`
}

type artistCredit struct {
	JoinPhrase string      `json:"joinphrase"`
	Artist     trackArtist `json:"artist"`
	Name       string      `json:"name"`
}

type attributeValues struct {
}

type attributeIDs struct {
}

type relation struct {
	AttributeValues attributeValues `json:"attribute-values"`
	Type            string          `json:"type"`
	TargetType      string          `json:"target-type"`
	Begin           string          `json:"begin"`
	Ended           bool            `json:"ended"`
	Artist          trackArtist     `json:"artist"`
	// TypeID          string          `json:"type-id"`
	End          string       `json:"end"`
	SourceCredit string       `json:"source-credit"`
	AttributeIDs attributeIDs `json:"attribute-ids"`
	TargetCredit string       `json:"target-credit"`
	Direction    string       `json:"direction"`
	Attributes   []string     `json:"attributes"`
}

type genre struct {
	Count int32  `json:"count"`
	Name  string `json:"name"`
}

type recording struct {
	Title     string     `json:"title"`
	Relations []relation `json:"relations"`
	// Disambiguation string     `json:"disambiguation"`
	ID     string  `json:"id"`
	Length int32   `json:"length"`
	Genres []genre `json:"genres"`
	// ArtistCredits  []artistCredit `json:"artist-credit"`
}

type track struct {
	Number    string    `json:"number"`
	Recording recording `json:"recording"`
	Title     string    `json:"title"`
	ID        string    `json:"id"`
	Length    int32     `json:"length"`
	Position  int32     `json:"position"`
	// ArtistCredits []artistCredit `json:"artist-credit"`
}

type mediaFullInfo struct {
	Position    int32   `json:"position"`
	TrackCount  int32   `json:"track-count"`
	TrackOffset int32   `json:"track-offset"`
	Tracks      []track `json:"tracks"`
	Title       string  `json:"title"`
	Format      string  `json:"format"`
}

type coverInfo struct {
	Images     []imageInfo `json:"images"`
	ReleaseURL string      `json:"release"`
}

type releaseGroup struct {
	Annotation       string `json:"annotation,omitempty"`
	FirstReleaseDate string `json:"first-release-date"`
	Title            string `json:"title"`
	ID               string `json:"ID"`
}

type releaseInfo struct {
	Asin               string             `json:"asin"`
	ArtistCredit       []artistCredit     `json:"artist-credit"`
	Barcode            string             `json:"barcode"`
	Title              string             `json:"title"`
	Media              []mediaFullInfo    `json:"media"`
	ID                 string             `json:"id"`
	Packaging          string             `json:"packaging"`
	Status             string             `json:"status"`
	Relations          []relation         `json:"relations"`
	Date               string             `json:"date"`
	Country            string             `json:"country"`
	LabelInfo          []labelInfo        `json:"label-info"`
	ReleaseGroup       releaseGroup       `json:"release-group"`
	Annotation         string             `json:"annotation"`
	TextRepresentation textRepresentation `json:"text-representation"`
}

type releaseSearchItem struct {
	ID    string `json:"id"`
	Score int32  `json:"score"`
	// Count     int32  `json:"count"`
	Title        string         `json:"title"`
	Status       string         `json:"status"`
	Packaging    string         `json:"packaging"`
	ArtistCredit []artistCredit `json:"artist-credit"`
	// ReleaseGroup ShortReleaseGroup `json:"release-group"`
	Date      string      `json:"date"`
	Country   string      `json:"country"`
	Barcode   string      `json:"barcode"`
	LabelInfo []labelInfo `json:"label-info"`
	// TrackCount int32       `json:"track-count"`
	Media              []media            `json:"media"`
	TextRepresentation textRepresentation `json:"text-representation"`
}

type releaseSearchResult struct {
	Created  string              `json:"created"`
	Count    int32               `json:"count"` // count of result items
	Offset   int32               `json:"offset"`
	Releases []releaseSearchItem `json:"releases"`
}

// Release converts data to common album format.
func (ri *releaseInfo) Release(r *md.Release) {
	r.Title = ri.Title
	// album.Record
	r.Country = ri.Country
	// album.Edition.ExtraInfo
	r.IDs[md.MusicbrainzAlbumID] = ri.ID
	if len(ri.Barcode) > 0 {
		r.Publishing.IDs[md.Barcode] = ri.Barcode
	}
	if len(ri.Asin) > 0 {
		r.IDs[md.Asin] = ri.Asin
	}
	r.Notes = ri.Annotation
	for _, li := range ri.LabelInfo {
		lbl := li.NewLabel()
		if !collection.Contains(lbl, r.Publishing.Labels) {
			r.Publishing.Labels = append(r.Publishing.Labels, lbl)
		}
	}
	r.Year = tp.NaiveStringToInt(ri.Date)
	ri.ReleaseGroup.ReleaseGroup(r)
	for i, mediaDisc := range ri.Media {
		disc := r.Disc(i + 1)
		for _, track := range mediaDisc.Tracks {
			tr := track.Track(disc)
			tr.Composition.Lyrics.Language = ri.TextRepresentation.Language
			for _, genre := range track.Recording.Genres {
				tr.Record.Genres = append(tr.Record.Genres, genre.Name)
			}
			r.Tracks = append(r.Tracks, tr)
			r.TotalTracks++
		}
		r.TotalDiscs++
	}
	for _, ac := range ri.ArtistCredit {
		ac.AddPerformer(r)
	}
	r.ReleaseStatus.Decode(ri.Status)
	// ri.Packaging
}

func (rgi releaseGroup) ReleaseGroup(r *md.Release) {
	r.Original.IDs[md.MusicbrainzReleaseGroupID] = rgi.ID
	r.Original.Year = tp.NaiveStringToInt(strings.SplitN(rgi.FirstReleaseDate, "-", 3)[0])
	if len(rgi.Annotation) > 0 {
		r.Original.Notes = rgi.Annotation
	}
}

func (rs releaseSearchResult) Search() []*md.Release {
	var ret []*md.Release
	var r *md.Release
	for _, searchItem := range rs.Releases {
		if r = searchItem.Release(); r != nil {
			ret = append(ret, r)
		}
	}
	return ret
}

func (si releaseSearchItem) Release() *md.Release {
	r := md.NewRelease()
	r.IDs[md.MusicbrainzAlbumID] = si.ID
	r.Title = si.Title
	if len(si.Barcode) > 0 {
		r.Publishing.IDs[md.Barcode] = si.Barcode
	}
	for _, li := range si.LabelInfo {
		r.Publishing.Labels = append(r.Publishing.Labels, li.NewLabel())
	}
	for _, ac := range si.ArtistCredit {
		ac.AddPerformer(r)
	}
	r.ReleaseStatus.Decode(si.Status)
	return r
}

// TODO: имеет смысл обработать другие типы картинок (например, "Medium")
func (ci coverInfo) Cover() *md.PictureInAudio {
	for _, imgInfo := range ci.Images {
		for _, imgType := range imgInfo.Types {
			if imgType == "Front" {
				ret := &md.PictureInAudio{
					PictType: md.PictTypeCoverFront,
					CoverURL: imgInfo.Thumbnails.URLLarge,
				}
				if len(imgInfo.Comment) > 0 {
					ret.Notes = imgInfo.Comment
				}
				return ret
			}
		}
	}
	return nil
}

func (li labelInfo) NewLabel() *md.Label {
	lbl := md.NewLabel(li.Label.Name, li.CatalogNumber)
	if li.CatalogNumber != "" {
		lbl.IDs[md.MusicbrainzLabelID] = li.Label.ID
	}
	return lbl
}

func (ac artistCredit) AddPerformer(r *md.Release) {
	if ac.Name != "" {
		r.Actors.Add(ac.Name, md.DiscogsArtistID, ac.Artist.ID)
		r.ActorRoles.Add(ac.Name, "performer")
	}
}

// TODO: добавить информацию о диске.
func (mi mediaFullInfo) Disc(disc *md.Disc) []*md.Track {
	var tracks []*md.Track
	for _, tr := range mi.Tracks {
		tracks = append(tracks, tr.Track(disc))
	}
	return tracks
}

func (tr *track) Track(disc *md.Disc) *md.Track {
	track := md.NewTrack()
	if disc != nil {
		track.LinkWithDisc(disc)
	}
	track.Position = strconv.Itoa(int(tr.Position))
	track.Title = tr.Title
	track.Duration = intutils.Duration(tr.Length)
	for _, rel := range tr.Recording.Relations {
		rel.AddActor(track)
	}
	return track
}

func (rel *relation) AddActor(track *md.Track) {
	if rel.Artist.Name != "" {
		var roles []string
		if rel.Type == "instrument" {
			roles = rel.Attributes
		} else {
			roles = []string{rel.Type}
		}
		for _, role := range roles {
			ActorsByRole(track, role).Add(rel.Artist.Name, role)
			track.Actors.Add(rel.Artist.Name, md.DiscogsArtistID, rel.Artist.ID)
		}
	}
}

// ActorsByRole определяет коллекцию для размещения описания по наименованию роли.
// Это может быть коллекция для описания акторов произведения, записи или релиза.
func ActorsByRole(track *md.Track, role string) *md.ActorRoles {
	switch role {
	case "design", "illustration", "design/illustration", "photography":
		return &track.ActorRoles
	// TODO: проверить!
	case "composer", "lyricist", "writer":
		return &track.Composition.ActorRoles
	default:
		return &track.Record.ActorRoles
	}
}
