package httpapi

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/alreadyonapp/alreadyon/server/internal/auth"
	"github.com/alreadyonapp/alreadyon/server/internal/config"
	"github.com/alreadyonapp/alreadyon/server/internal/library"
	"github.com/alreadyonapp/alreadyon/server/internal/playback"
	"github.com/alreadyonapp/alreadyon/server/internal/schedule"
)

// TEMPORARY testing tool: remove this handler and player control before release.
// Management authentication represents the installation's administrator account.
// IDs and the presented airing must agree; clients never supply filesystem paths.
func (p *PlaybackHTTP) deleteCommercial(w http.ResponseWriter, r *http.Request) {
	if auth.PrincipalID(r.Context()) == 0 {
		writeJSONError(w, 401, "sign in required")
		return
	}
	var body struct {
		ChannelID     int64     `json:"channelId"`
		MediaFileID   int64     `json:"mediaFileId"`
		EntryStartsAt time.Time `json:"entryStartsAt"`
	}
	if !decodeStrict(w, r, &body) {
		return
	}
	entry, _, err := schedule.Current(r.Context(), p.DB, schedule.FixedClock{Time: p.Clock.Now().Add(-playback.PresentationDelay)}, body.ChannelID)
	if err != nil || entry.Kind != schedule.KindCommercial || entry.MediaFileID != body.MediaFileID || !entry.StartsAt.Equal(body.EntryStartsAt) {
		writeJSONError(w, 409, "That commercial is no longer airing. Nothing was deleted.")
		return
	}
	var path string
	if err := p.DB.QueryRowContext(r.Context(), `SELECT path FROM media_files WHERE id=? AND root_kind='commercial' AND status='ok'`, body.MediaFileID).Scan(&path); err != nil {
		writeJSONError(w, 409, "Commercial is no longer available.")
		return
	}
	cfg, err := library.LoadScanConfig(r.Context(), p.DB)
	if err != nil {
		writeJSONError(w, 500, "Could not load commercial folders.")
		return
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		writeJSONError(w, 409, "Commercial file is unavailable.")
		return
	}
	for _, root := range cfg.Roots {
		if root.Kind != config.RootKindCommercial || !config.PathWithinRoot(root, real) {
			continue
		}
		// OpenRoot confines traversal even if an ancestor is replaced by a symlink.
		dir, err := os.OpenRoot(root.RealPath)
		if err != nil {
			continue
		}
		defer dir.Close()
		rel, err := filepath.Rel(root.RealPath, real)
		if err != nil {
			continue
		}
		info, err := dir.Lstat(rel)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if err := removeCommercial(r, p.DB, dir, rel, body.MediaFileID); err != nil {
			writeJSONError(w, 500, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSONError(w, 403, "File is outside the configured commercial folders.")
}

// Mark missing in the same transaction as removal so failed filesystem writes
// leave the catalog usable. Keep schedule/history references until they expire.
func removeCommercial(r *http.Request, db *sql.DB, root *os.Root, rel string, id int64) error {
	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		return errors.New("Could not update the commercial catalog.")
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(r.Context(), `UPDATE media_files SET status='missing' WHERE id=? AND root_kind='commercial' AND status='ok'`, id)
	if err != nil {
		return errors.New("Could not update the commercial catalog.")
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return errors.New("Commercial is no longer available.")
	}
	if err := root.Remove(rel); err != nil {
		return errors.New("Could not delete commercial. Check folder write permissions and read-only mounts.")
	}
	if err := tx.Commit(); err != nil {
		return errors.New("File deleted, but catalog update failed. Rescan commercial folders.")
	}
	return nil
}
