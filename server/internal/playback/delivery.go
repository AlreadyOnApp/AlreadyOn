package playback

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
)

type Delivery struct {
	// MediaURL is retained for API/storage compatibility but no longer selects
	// another origin: UI, API and playback share the request origin.
	MediaURL       string   `json:"mediaUrl"`
	AllowedOrigins []string `json:"allowedOrigins"`
}

func (d Delivery) Validate() error {
	if len(d.AllowedOrigins) > 16 {
		return errors.New("playback: too many allowed origins")
	}
	values := append([]string{}, d.AllowedOrigins...)
	if d.MediaURL != "" {
		values = append(values, d.MediaURL)
	}
	for _, raw := range values {
		u, err := url.Parse(raw)
		if err != nil || len(raw) > 512 || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return errors.New("playback: delivery URLs must be HTTP(S) origins without credentials, paths or queries")
		}
	}
	return nil
}
func GetDelivery(ctx context.Context, db *sql.DB) (Delivery, error) {
	var d Delivery
	var raw string
	err := db.QueryRowContext(ctx, `SELECT media_url,allowed_origins FROM playback_delivery WHERE id=1`).Scan(&d.MediaURL, &raw)
	if err == nil {
		err = json.Unmarshal([]byte(raw), &d.AllowedOrigins)
	}
	return d, err
}
func SaveDelivery(ctx context.Context, db *sql.DB, d Delivery) error {
	if err := d.Validate(); err != nil {
		return err
	}
	if d.AllowedOrigins == nil {
		d.AllowedOrigins = []string{}
	}
	raw, err := json.Marshal(d.AllowedOrigins)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `UPDATE playback_delivery SET media_url=?,allowed_origins=? WHERE id=1`, d.MediaURL, string(raw))
	return err
}
