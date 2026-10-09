package channel

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestFilterNormalize(t *testing.T) {
	ok := Filter{AudienceTier: "teen"}
	n, err := ok.Normalize()
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if n.MediaType != "both" || n.Era != "both" || n.Genres == nil {
		t.Errorf("defaults not applied: %+v", n)
	}

	n, err = Filter{AudienceTier: "adult", Genres: []string{"comedy", "drama", "comedy"}, TitleContains: "  Harry  "}.Normalize()
	if err != nil || len(n.Genres) != 2 || n.TitleContains != "Harry" {
		t.Errorf("Normalize = %+v, %v; want de-duplicated genres and a trimmed title", n, err)
	}

	cases := map[string]struct {
		f    Filter
		want error
	}{
		"missing tier":     {Filter{}, ErrInvalidAudienceTier},
		"bad tier":         {Filter{AudienceTier: "kid"}, ErrInvalidAudienceTier},
		"unknown genre":    {Filter{AudienceTier: "adult", Genres: []string{"comedy", "nope"}}, ErrInvalidGenre},
		"bad media type":   {Filter{AudienceTier: "adult", MediaType: "radio"}, ErrInvalidMediaType},
		"bad era":          {Filter{AudienceTier: "adult", Era: "old"}, ErrInvalidEra},
		"title too long":   {Filter{AudienceTier: "adult", TitleContains: strings.Repeat("a", MaxTitleFilterLength+1)}, ErrInvalidTitleFilter},
		"control in title": {Filter{AudienceTier: "adult", TitleContains: "a\nb"}, ErrInvalidTitleFilter},
	}
	for name, c := range cases {
		if _, err := c.f.Normalize(); !errors.Is(err, c.want) {
			t.Errorf("%s: err = %v, want %v", name, err, c.want)
		}
	}
}

func TestSetGetClearFilter(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	c, _ := Create(ctx, db, Draft{Number: 200, Name: "Custom"})

	if f, err := GetFilter(ctx, db, c.ID); err != nil || f != nil {
		t.Fatalf("GetFilter before set = %v, %v; want nil", f, err)
	}
	saved, err := SetFilter(ctx, db, c.ID, Filter{AudienceTier: "teen", Genres: []string{"comedy"}, MediaType: "movie", Era: "nostalgia", TitleContains: "Potter"})
	if err != nil {
		t.Fatalf("SetFilter: %v", err)
	}
	got, err := GetFilter(ctx, db, c.ID)
	if err != nil || got == nil || got.MediaType != "movie" || got.Era != "nostalgia" || got.TitleContains != "Potter" || len(got.Genres) != 1 || saved.AudienceTier != "teen" {
		t.Fatalf("GetFilter = %+v, %v", got, err)
	}

	// Saving again replaces rather than duplicates.
	if _, err := SetFilter(ctx, db, c.ID, Filter{AudienceTier: "adult"}); err != nil {
		t.Fatalf("SetFilter (update): %v", err)
	}
	all, _ := ListFilters(ctx, db)
	if len(all) != 1 || all[c.ID].AudienceTier != "adult" || len(all[c.ID].Genres) != 0 {
		t.Errorf("ListFilters = %+v, want one replaced filter", all)
	}

	if err := ClearFilter(ctx, db, c.ID); err != nil {
		t.Fatalf("ClearFilter: %v", err)
	}
	if f, _ := GetFilter(ctx, db, c.ID); f != nil {
		t.Errorf("filter still present after ClearFilter: %+v", f)
	}
}

func TestSetFilter_BuiltInChannelsCanHaveOneButMissingChannelsCannot(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	var builtInID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO channels (number, name) VALUES (150, 'Built-in') RETURNING id`).Scan(&builtInID); err != nil {
		t.Fatalf("insert built-in channel: %v", err)
	}
	if _, err := SetFilter(ctx, db, builtInID, Filter{AudienceTier: "adult"}); err != nil {
		t.Errorf("built-in: err = %v, want nil", err)
	}
	if _, err := SetFilter(ctx, db, 99999, Filter{AudienceTier: "adult"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing: err = %v, want ErrNotFound", err)
	}
}

func TestFilter_DeletedWithItsChannel(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	c, _ := Create(ctx, db, Draft{Number: 200, Name: "Custom"})
	SetFilter(ctx, db, c.ID, Filter{AudienceTier: "adult"})
	if err := Delete(ctx, db, c.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if all, _ := ListFilters(ctx, db); len(all) != 0 {
		t.Errorf("filter survived its channel: %+v", all)
	}
}
