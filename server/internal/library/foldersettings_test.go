package library

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestFolderSettings_FreshDatabase_UsesDefaults(t *testing.T) {
	db := openClassifyTestDB(t)
	s, err := GetFolderSettings(context.Background(), db)
	if err != nil {
		t.Fatalf("GetFolderSettings: %v", err)
	}
	if s.TV != nil || s.Movie != nil || s.Commercial != nil || s.Exclusions != nil {
		t.Fatalf("fresh settings = %#v, want all nil (not set)", s)
	}
	e := s.Effective()
	if !reflect.DeepEqual(e.TV.Values, []string{"/data/media/tv_shows"}) ||
		!reflect.DeepEqual(e.Movie.Values, []string{"/data/media/movies"}) ||
		!reflect.DeepEqual(e.Commercial.Values, []string{"/data/media/commercials"}) {
		t.Errorf("default paths = %v %v %v", e.TV.Values, e.Movie.Values, e.Commercial.Values)
	}
	if e.TV.Source != SourceDefault || e.Exclusions.Source != SourceDefault {
		t.Errorf("sources = %s / %s, want default", e.TV.Source, e.Exclusions.Source)
	}
}

func TestFolderSettings_SaveAndReadBack_AdminOverridesDefault(t *testing.T) {
	db := openClassifyTestDB(t)
	ctx := context.Background()
	saved, err := SaveFolderSettings(ctx, db, FolderSettings{
		TV:    []string{"/srv/media/TV Shows", " /extra/tv/ ", "/srv/media/TV Shows"},
		Movie: []string{},
	})
	if err != nil {
		t.Fatalf("SaveFolderSettings: %v", err)
	}
	if want := []string{"/srv/media/TV Shows", "/extra/tv"}; !reflect.DeepEqual(saved.TV, want) {
		t.Errorf("saved TV = %v, want %v (trimmed, cleaned, de-duplicated)", saved.TV, want)
	}

	got, err := GetFolderSettings(ctx, db)
	if err != nil {
		t.Fatalf("GetFolderSettings: %v", err)
	}
	e := got.Effective()
	if e.TV.Source != SourceAdmin || !reflect.DeepEqual(e.TV.Values, saved.TV) {
		t.Errorf("TV effective = %+v", e.TV)
	}
	if e.Movie.Source != SourceAdmin || len(e.Movie.Values) != 0 {
		t.Errorf("empty movie list = %+v, want admin-set and empty (not the default)", e.Movie)
	}
	if e.Commercial.Source != SourceDefault {
		t.Errorf("untouched commercial source = %s, want default", e.Commercial.Source)
	}
}

func TestFolderSettings_NilResetsToDefault(t *testing.T) {
	db := openClassifyTestDB(t)
	ctx := context.Background()
	if _, err := SaveFolderSettings(ctx, db, FolderSettings{TV: []string{"/a"}}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := SaveFolderSettings(ctx, db, FolderSettings{}); err != nil {
		t.Fatalf("save reset: %v", err)
	}
	got, _ := GetFolderSettings(ctx, db)
	if got.TV != nil {
		t.Errorf("TV = %v after reset, want nil", got.TV)
	}
}

func TestFolderSettings_Validation(t *testing.T) {
	db := openClassifyTestDB(t)
	ctx := context.Background()
	tooMany := make([]string, maxFolderPaths+1)
	for i := range tooMany {
		tooMany[i] = "/p" + string(rune('a'+i%26)) + string(rune('a'+i/26))
	}
	cases := []struct {
		name string
		s    FolderSettings
		want error
	}{
		{"relative path", FolderSettings{TV: []string{"tv/shows"}}, ErrInvalidFolderPath},
		{"empty path", FolderSettings{Movie: []string{"  "}}, ErrInvalidFolderPath},
		{"NUL byte", FolderSettings{Commercial: []string{"/a\x00b"}}, ErrInvalidFolderPath},
		{"too many folders", FolderSettings{TV: tooMany}, ErrTooManyEntries},
		{"bad glob", FolderSettings{Exclusions: []string{"[unclosed"}}, ErrInvalidPattern},
		{"empty pattern", FolderSettings{Exclusions: []string{""}}, ErrInvalidPattern},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := SaveFolderSettings(ctx, db, tc.s); !errors.Is(err, tc.want) {
				t.Errorf("err = %v, want %v", err, tc.want)
			}
		})
	}
	// A rejected save must not have changed anything.
	if got, _ := GetFolderSettings(ctx, db); got.TV != nil || got.Exclusions != nil {
		t.Errorf("settings changed by rejected saves: %#v", got)
	}
}

func TestEffectiveFolders_ValidateRoots_MergesTVAndMoviesOnce(t *testing.T) {
	shared, tv, commercial := t.TempDir(), t.TempDir(), t.TempDir()
	e := FolderSettings{
		TV:         []string{tv, shared},
		Movie:      []string{shared, "/definitely/not/here"},
		Commercial: []string{commercial},
	}.Effective()

	roots, errs := e.ValidateRoots()
	if len(errs) != 1 {
		t.Fatalf("errs = %v, want exactly the one missing folder", errs)
	}
	media, commercials := 0, 0
	for _, r := range roots {
		switch r.Kind {
		case "media":
			media++
		case "commercial":
			commercials++
		}
	}
	if media != 2 || commercials != 1 {
		t.Errorf("roots: media=%d commercial=%d, want 2 media (shared folder once) and 1 commercial", media, commercials)
	}
}
