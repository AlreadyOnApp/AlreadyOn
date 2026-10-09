package channel

import (
	"context"
	"testing"
)

func TestSeedDefaults_PopulatesEmptyTable(t *testing.T) {
	db := openTestDB(t)
	n, err := SeedDefaults(context.Background(), db)
	if err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}
	if n != len(defaultChannels) {
		t.Errorf("inserted = %d, want %d", n, len(defaultChannels))
	}

	channels, err := List(context.Background(), db)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(channels) != len(defaultChannels) {
		t.Fatalf("len(channels) = %d, want %d", len(channels), len(defaultChannels))
	}
	for _, c := range channels {
		if !isReservedNumber(c.Number) {
			t.Errorf("seeded channel %q has number %d, want it within %d-%d",
				c.Name, c.Number, DefaultChannelRangeStart, DefaultChannelRangeEnd)
		}
	}
}

func TestSeedDefaults_SkipsWhenTableAlreadyHasRows(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	if _, err := Create(ctx, db, Draft{Number: 1000, Name: "Pre-existing custom channel"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	n, err := SeedDefaults(ctx, db)
	if err != nil {
		t.Fatalf("SeedDefaults: %v", err)
	}
	if n != 0 {
		t.Errorf("inserted = %d, want 0 (should skip a non-empty table)", n)
	}

	channels, err := List(ctx, db)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(channels) != 1 {
		t.Errorf("len(channels) = %d, want 1 (only the pre-existing channel)", len(channels))
	}
}

func TestSeedDefaults_IsIdempotentAcrossRestarts(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()
	if _, err := SeedDefaults(ctx, db); err != nil {
		t.Fatalf("first SeedDefaults: %v", err)
	}
	if _, err := SeedDefaults(ctx, db); err != nil {
		t.Fatalf("second SeedDefaults: %v", err)
	}

	channels, err := List(ctx, db)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(channels) != len(defaultChannels) {
		t.Errorf("len(channels) = %d, want %d (should not duplicate on a second call)", len(channels), len(defaultChannels))
	}
}

func TestSeedDefaults_NoDuplicateNumbers(t *testing.T) {
	seen := make(map[int]bool)
	for _, d := range defaultChannels {
		if seen[d.Number] {
			t.Errorf("duplicate default channel number %d", d.Number)
		}
		seen[d.Number] = true
	}
}
