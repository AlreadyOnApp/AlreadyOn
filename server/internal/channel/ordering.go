package channel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
)

// LineupItem is one piece of library content attached to a channel, in
// play order.
type LineupItem struct {
	ID          int64
	ChannelID   int64
	MediaFileID int64
	Position    int
}

// EpisodeInfo is the ordering-relevant subset of an identified library
// item (see library.Identification), used to sort a show's episodes
// into broadcast order before attaching them to a channel.
type EpisodeInfo struct {
	MediaFileID int64
	Season      int
	Episode     int
}

var (
	ErrMediaFileNotFound = errors.New("channel: media file not found")
	ErrDuplicateItem     = errors.New("channel: media file already in this channel's lineup")
)

// SortEpisodes returns a new slice ordered by season then episode number
// — the sequential order episodes air/play in. It's a stable sort, so
// two items that (incorrectly) claim the same season/episode keep their
// original relative order rather than depending on map/query iteration
// order — what makes the result deterministic across repeated calls,
// not just "sorted".
func SortEpisodes(items []EpisodeInfo) []EpisodeInfo {
	sorted := make([]EpisodeInfo, len(items))
	copy(sorted, items)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Season != sorted[j].Season {
			return sorted[i].Season < sorted[j].Season
		}
		return sorted[i].Episode < sorted[j].Episode
	})
	return sorted
}

// ListItems returns channelID's current lineup, in play order. Gaps
// between position values (left behind by RemoveItem) don't affect the
// result — the order comes from ORDER BY position, not from positions
// being contiguous.
func ListItems(ctx context.Context, db *sql.DB, channelID int64) ([]LineupItem, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, channel_id, media_file_id, position FROM channel_items
		WHERE channel_id = ? ORDER BY position`, channelID)
	if err != nil {
		return nil, fmt.Errorf("channel: list items for channel %d: %w", channelID, err)
	}
	defer rows.Close()

	var items []LineupItem
	for rows.Next() {
		var it LineupItem
		if err := rows.Scan(&it.ID, &it.ChannelID, &it.MediaFileID, &it.Position); err != nil {
			return nil, fmt.Errorf("channel: list items for channel %d: %w", channelID, err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("channel: list items for channel %d: %w", channelID, err)
	}
	return items, nil
}

// AppendItems adds mediaFileIDs to the end of channelID's lineup, in
// the given order — callers wanting episodes in broadcast order should
// sort with SortEpisodes first and pass the result's MediaFileIDs.
// Fails atomically (no partial append) if the channel doesn't exist,
// any media file doesn't exist, or any media file is already in this
// channel's lineup.
func AppendItems(ctx context.Context, db *sql.DB, channelID int64, mediaFileIDs []int64) ([]LineupItem, error) {
	if len(mediaFileIDs) == 0 {
		return nil, nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("channel: append items to channel %d: %w", channelID, err)
	}
	defer tx.Rollback()

	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM channels WHERE id = ?)`, channelID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("channel: append items to channel %d: %w", channelID, err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	var nextPosition int
	err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(position) + 1, 0) FROM channel_items WHERE channel_id = ?`, channelID).
		Scan(&nextPosition)
	if err != nil {
		return nil, fmt.Errorf("channel: append items to channel %d: %w", channelID, err)
	}

	appended := make([]LineupItem, 0, len(mediaFileIDs))
	for i, mediaFileID := range mediaFileIDs {
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM media_files WHERE id = ?)`, mediaFileID).Scan(&exists); err != nil {
			return nil, fmt.Errorf("channel: append items to channel %d: %w", channelID, err)
		}
		if !exists {
			return nil, ErrMediaFileNotFound
		}

		var alreadyPresent bool
		err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM channel_items WHERE channel_id = ? AND media_file_id = ?)`,
			channelID, mediaFileID).Scan(&alreadyPresent)
		if err != nil {
			return nil, fmt.Errorf("channel: append items to channel %d: %w", channelID, err)
		}
		if alreadyPresent {
			return nil, ErrDuplicateItem
		}

		position := nextPosition + i
		res, err := tx.ExecContext(ctx, `
			INSERT INTO channel_items (channel_id, media_file_id, position) VALUES (?, ?, ?)`,
			channelID, mediaFileID, position)
		if err != nil {
			return nil, fmt.Errorf("channel: append items to channel %d: %w", channelID, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("channel: append items to channel %d: %w", channelID, err)
		}
		appended = append(appended, LineupItem{ID: id, ChannelID: channelID, MediaFileID: mediaFileID, Position: position})
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("channel: append items to channel %d: %w", channelID, err)
	}
	return appended, nil
}

// RemoveItem removes one item from channelID's lineup by its lineup
// item id. Returns ErrNotFound if it isn't part of that channel's
// lineup. Leaves a gap in position rather than renumbering the rest of
// the lineup — ListItems's ORDER BY position is unaffected by gaps, and
// future schedule generation reads the lineup by its result-set index,
// not by raw position values.
func RemoveItem(ctx context.Context, db *sql.DB, channelID, itemID int64) error {
	res, err := db.ExecContext(ctx, `DELETE FROM channel_items WHERE id = ? AND channel_id = ?`, itemID, channelID)
	if err != nil {
		return fmt.Errorf("channel: remove item %d from channel %d: %w", itemID, channelID, err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("channel: remove item %d from channel %d: %w", itemID, channelID, err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ClearItems removes every item from channelID's lineup.
func ClearItems(ctx context.Context, db *sql.DB, channelID int64) error {
	if _, err := db.ExecContext(ctx, `DELETE FROM channel_items WHERE channel_id = ?`, channelID); err != nil {
		return fmt.Errorf("channel: clear items for channel %d: %w", channelID, err)
	}
	return nil
}
