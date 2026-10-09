package metadata

import (
	"context"
	"fmt"
)

type DirectDetails struct {
	Title         string
	Tags          []Tag
	Overview      string
	Date          string
	LastDate      string
	Certification string
}

func (c *Client) LookupID(ctx context.Context, mediaType string, id int) (DirectDetails, error) {
	if !c.enabled() || id <= 0 {
		return DirectDetails{}, fmt.Errorf("TMDb lookup requires a configured API token and positive ID")
	}
	kind := "movie"
	if mediaType == "episode" {
		kind = "tv"
	}
	var raw struct {
		Title    string `json:"title"`
		Name     string `json:"name"`
		Language string `json:"original_language"`
		Genres   []struct {
			ID int `json:"id"`
		} `json:"genres"`
	}
	if err := c.get(ctx, fmt.Sprintf("/%s/%d", kind, id), &raw); err != nil {
		return DirectDetails{}, err
	}
	ids := []int{}
	for _, g := range raw.Genres {
		ids = append(ids, g.ID)
	}
	genreMap := movieGenreTags
	if kind == "tv" {
		genreMap = tvGenreTags
	}
	d := DirectDetails{Title: raw.Title, Tags: tagsFromGenreIDs(genreMap, ids)}
	if kind == "tv" {
		d.Title = raw.Name
		if raw.Language == "ja" {
			for _, tag := range d.Tags {
				if tag == TagAnimation {
					d.Tags = append(d.Tags, TagAnime)
					break
				}
			}
		}
		detail, err := c.TVDetails(ctx, id)
		if err != nil {
			return d, err
		}
		d.Overview = detail.Overview
		d.Date = detail.FirstAirDate
		d.LastDate = detail.LastAirDate
		d.Certification = detail.Certification
	} else {
		detail, err := c.MovieDetails(ctx, id)
		if err != nil {
			return d, err
		}
		d.Overview = detail.Overview
		d.Date = detail.ReleaseDate
		d.Certification = detail.Certification
	}
	if d.Title == "" {
		return d, fmt.Errorf("TMDb returned no title for that ID")
	}
	return d, nil
}
