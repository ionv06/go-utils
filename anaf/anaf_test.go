package anaf

import (
	"errors"
	"testing"
)

func TestParseFiscalCode(t *testing.T) {
	cases := []struct {
		in      string
		wantCUI int
		wantErr error
	}{
		// Real, checksum-correct codes, with the accepted input spellings.
		{"1590082", 1590082, nil},
		{"RO1590082", 1590082, nil},
		{" ro 1.590.082 ", 1590082, nil},
		{"14399840", 14399840, nil},

		{"", 0, ErrMissingCode},
		{"RO", 0, ErrMissingCode},
		{"1590083", 0, ErrInvalidCode},     // control digit off by one
		{"1", 0, ErrInvalidCode},           // too short
		{"12345678901", 0, ErrInvalidCode}, // too long
	}
	for _, c := range cases {
		cui, _, err := parseFiscalCode(c.in)
		if !errors.Is(err, c.wantErr) {
			t.Errorf("parseFiscalCode(%q) error = %v, want %v", c.in, err, c.wantErr)
			continue
		}
		if err == nil && cui != c.wantCUI {
			t.Errorf("parseFiscalCode(%q) = %d, want %d", c.in, cui, c.wantCUI)
		}
	}
}

func TestNormalizeLocality(t *testing.T) {
	cases := []struct{ in, wantLoc, wantSector string }{
		{"Ors. Voluntari", "Voluntari", ""},
		{"Sector 3 BUCURESTI", "BUCURESTI", "3"},
		{"Cluj-Napoca", "Cluj-Napoca", ""},
	}
	for _, c := range cases {
		loc, sector := normalizeLocality(c.in)
		if loc != c.wantLoc || sector != c.wantSector {
			t.Errorf("normalizeLocality(%q) = (%q,%q), want (%q,%q)", c.in, loc, sector, c.wantLoc, c.wantSector)
		}
	}
}
