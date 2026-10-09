package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestServerSettings_DefaultsThenPartialUpdate(t *testing.T) {
	router := NewRouter(t.TempDir(), t.TempDir(), openTestDB(t))

	rec := doJSON(t, router, http.MethodGet, "/api/settings/server", "")
	var s serverSettingsResponse
	json.Unmarshal(rec.Body.Bytes(), &s)
	want := serverSettingsResponse{"software", "self-signed", "letsencrypt", "", "none", false}
	if rec.Code != http.StatusOK || s != want {
		t.Fatalf("defaults = %d %+v, want %+v", rec.Code, s, want)
	}

	rec = doJSON(t, router, http.MethodPut, "/api/settings/server", `{"tlsMode":"acme","acmeDomain":"tv.example.com"}`)
	json.Unmarshal(rec.Body.Bytes(), &s)
	if rec.Code != http.StatusOK || s.TLSMode != "acme" || s.ACMEDomain != "tv.example.com" || s.Transcoding != "software" {
		t.Fatalf("after partial PUT = %d %+v", rec.Code, s)
	}

	rec = doJSON(t, router, http.MethodGet, "/api/settings/server", "")
	var again serverSettingsResponse
	json.Unmarshal(rec.Body.Bytes(), &again)
	if again != s {
		t.Errorf("GET after PUT = %+v, want the saved %+v", again, s)
	}
}

func TestServerSettings_RejectsInvalidValues(t *testing.T) {
	router := NewRouter(t.TempDir(), t.TempDir(), openTestDB(t))
	for _, body := range []string{
		`{"transcoding":"quantum"}`,
		`{"tlsMode":"none"}`,
		`{"acmeProvider":"nobody"}`,
		`{"acmeDomain":"not a host"}`,
		`{"acmeDomain":"-bad.example.com"}`,
		`{"ddnsProvider":"godaddy"}`,
		`{"hardwareDecoding":true}`, // needs a hardware encoder first
		`{"ddnsApiToken":"secret"}`, // credentials are not stored here
		`nope`,
	} {
		if rec := doJSON(t, router, http.MethodPut, "/api/settings/server", body); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", body, rec.Code)
		}
	}
	rec := doJSON(t, router, http.MethodGet, "/api/settings/server", "")
	var s serverSettingsResponse
	json.Unmarshal(rec.Body.Bytes(), &s)
	if s.Transcoding != "software" || s.TLSMode != "self-signed" {
		t.Errorf("a rejected PUT changed the settings: %+v", s)
	}
}

func TestServerSettings_HardwareDecodingNeedsAHardwareEncoder(t *testing.T) {
	router := NewRouter(t.TempDir(), t.TempDir(), openTestDB(t))
	get := func() serverSettingsResponse {
		var s serverSettingsResponse
		json.Unmarshal(doJSON(t, router, http.MethodGet, "/api/settings/server", "").Body.Bytes(), &s)
		return s
	}
	if rec := doJSON(t, router, http.MethodPut, "/api/settings/server", `{"hardwareDecoding":true}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("decoding with a software encoder: %d, want 400", rec.Code)
	}
	if rec := doJSON(t, router, http.MethodPut, "/api/settings/server", `{"transcoding":"intel","hardwareDecoding":true}`); rec.Code != http.StatusOK {
		t.Fatalf("both together: %d %s", rec.Code, rec.Body.String())
	}
	if s := get(); s.Transcoding != "intel" || !s.HardwareDecoding {
		t.Fatalf("not saved: %+v", s)
	}
	// Hardware to hardware keeps decoding; software turns it off.
	doJSON(t, router, http.MethodPut, "/api/settings/server", `{"transcoding":"amd-vaapi"}`)
	if s := get(); !s.HardwareDecoding {
		t.Errorf("hardware to hardware should keep decoding: %+v", s)
	}
	doJSON(t, router, http.MethodPut, "/api/settings/server", `{"transcoding":"software"}`)
	if s := get(); s.HardwareDecoding || s.Transcoding != "software" {
		t.Errorf("software should turn decoding off: %+v", s)
	}
	if rec := doJSON(t, router, http.MethodPut, "/api/settings/server", `{"transcoding":"software","hardwareDecoding":true}`); rec.Code != http.StatusBadRequest {
		t.Errorf("asking for both software and decoding: %d, want 400", rec.Code)
	}
	// The web page sends every field back on Save.
	full := `{"transcoding":"intel","tlsMode":"self-signed","acmeProvider":"letsencrypt","acmeDomain":"","ddnsProvider":"none","hardwareDecoding":true}`
	if rec := doJSON(t, router, http.MethodPut, "/api/settings/server", full); rec.Code != http.StatusOK {
		t.Errorf("full-body save: %d %s", rec.Code, rec.Body.String())
	}
}
