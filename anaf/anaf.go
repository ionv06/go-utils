// Package anaf downloads Romanian company data from the ANAF public VAT
// registry web service.
//
// The entry point is Lookup: it takes a single Romanian fiscal code (CUI),
// validates it, calls ANAF and returns the normalized company data encoded as
// JSON. Callers must not pre-validate the code - Lookup owns that step and
// reports the outcome through the sentinel errors below.
package anaf

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	// Embeds Mozilla's CA root bundle in the binary as a fallback. Go only uses
	// it when the host has no system trust store - e.g. a scratch/alpine Docker
	// image without the ca-certificates package, which otherwise fails every
	// HTTPS call with "x509: certificate signed by unknown authority".
	// Certificate verification stays fully enabled.
	_ "golang.org/x/crypto/x509roots/fallback"
)

const vatURL = "https://webservicesp.anaf.ro/api/PlatitorTvaRest/v9/tva"

// httpTimeout bounds a single ANAF call end to end.
const httpTimeout = 20 * time.Second

var nonDigitRE = regexp.MustCompile(`\D+`)

// Sentinel errors describing an input or lookup outcome the caller may act on.
// Their text is safe to show to an end user; see UserMessage.
var (
	// ErrMissingCode means no fiscal code was supplied.
	ErrMissingCode = UserError{"cod fiscal lipsa"}
	// ErrInvalidCode means the code is not a well-formed Romanian CUI.
	ErrInvalidCode = UserError{"cod fiscal invalid"}
	// ErrNotFound means ANAF knows no company with this code.
	ErrNotFound = UserError{"cod fiscal negasit la ANAF"}
)

// UserError marks an error whose text is safe to show to the end user: bad
// input or a company ANAF does not know. Everything else - network, TLS,
// decode - is an internal fault and must not reach the browser verbatim.
type UserError struct{ Msg string }

func (e UserError) Error() string { return e.Msg }

// UserMessage returns the user-safe text of err, if it has one.
func UserMessage(err error) (string, bool) {
	var ue UserError
	if errors.As(err, &ue) {
		return ue.Msg, true
	}
	return "", false
}

// Company is the normalized subset of the ANAF record. Field names match the
// JSON keys produced by Lookup.
type Company struct {
	FiscalCode      string `json:"fiscal_code"`
	Name            string `json:"name"`
	TradeRegisterNo string `json:"trade_register_no"`
	Country         string `json:"country"`
	County          string `json:"county"`
	Locality        string `json:"locality"`
	Street          string `json:"street"`
	Number          string `json:"number"`
	Sector          string `json:"sector,omitempty"`
	Attribute       string `json:"attribute,omitempty"`
}

type request struct {
	CUI  int    `json:"cui"`
	Data string `json:"data"`
}

type response struct {
	Found []found `json:"found"`
}

type found struct {
	DateGenerale struct {
		CUI        json.Number `json:"cui"`
		NrRegCom   string      `json:"nrRegCom"`
		Denumire   string      `json:"denumire"`
		Adresa     string      `json:"adresa"`
		Data       string      `json:"data"`
		Stare      string      `json:"stare_inregistrare"`
		CodCAEN    string      `json:"cod_CAEN"`
		Organ      string      `json:"organFiscalCompetent"`
		ROEFactura bool        `json:"statusRO_e_Factura"`
	} `json:"date_generale"`
	InregistrareScopTVA struct {
		ScpTVA bool `json:"scpTVA"`
	} `json:"inregistrare_scop_Tva"`
	AdresaSediuSocial struct {
		Strada     string `json:"sdenumire_Strada"`
		Numar      string `json:"snumar_Strada"`
		Localitate string `json:"sdenumire_Localitate"`
		Judet      string `json:"sdenumire_Judet"`
		Detalii    string `json:"sdetalii_Adresa"`
		CodPostal  string `json:"scod_Postal"`
	} `json:"adresa_sediu_social"`
}

// Lookup validates the Romanian fiscal code, downloads the company data from
// ANAF and returns it as JSON (the encoding of Company).
//
// It returns ErrMissingCode, ErrInvalidCode or ErrNotFound for input and
// lookup problems; any other error is an internal fault (network, TLS, decode)
// whose text should be logged, not shown to the user.
func Lookup(fiscalCode string) ([]byte, error) {
	company, err := LookupCompany(fiscalCode)
	if err != nil {
		return nil, err
	}
	return json.Marshal(company)
}

// LookupCompany is Lookup without the JSON encoding step, for callers that
// want the struct directly.
func LookupCompany(fiscalCode string) (*Company, error) {
	cui, cuiText, err := parseFiscalCode(fiscalCode)
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal([]request{{
		CUI:  cui,
		Data: time.Now().Format("2006-01-02"),
	}})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, vatURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: httpTimeout}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("ANAF a raspuns cu status %s", res.Status)
	}

	dec := json.NewDecoder(res.Body)
	dec.UseNumber()
	var out response
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Found) == 0 {
		return nil, ErrNotFound
	}
	return companyFrom(out.Found[0], cuiText), nil
}

// parseFiscalCode normalizes the code and checks it is a valid Romanian CUI:
// 2..10 digits with a correct control digit. It returns the numeric CUI (the
// form ANAF expects) and its canonical digits-only text.
func parseFiscalCode(fiscalCode string) (int, string, error) {
	cuiText := CleanFiscalCode(fiscalCode)
	if cuiText == "" {
		return 0, "", ErrMissingCode
	}
	if len(cuiText) < 2 || len(cuiText) > 10 {
		return 0, "", ErrInvalidCode
	}
	cui, err := strconv.Atoi(cuiText)
	if err != nil || cui <= 0 {
		return 0, "", ErrInvalidCode
	}
	if !validControlDigit(cuiText) {
		return 0, "", ErrInvalidCode
	}
	return cui, cuiText, nil
}

// validControlDigit applies the official CUI checksum: the digits before the
// last one are weighted by 7,5,3,2,1,7,5,3,2 (right-aligned), the sum is
// multiplied by 10 and reduced modulo 11, and 10 folds back to 0.
func validControlDigit(digits string) bool {
	weights := []int{7, 5, 3, 2, 1, 7, 5, 3, 2}
	body := digits[:len(digits)-1]
	control := int(digits[len(digits)-1] - '0')

	sum := 0
	// Right-align the body against the weights so the last body digit gets the
	// weight 2, the one before it 3, and so on.
	offset := len(weights) - len(body)
	for i := 0; i < len(body); i++ {
		sum += int(body[i]-'0') * weights[offset+i]
	}
	expected := (sum * 10) % 11
	if expected == 10 {
		expected = 0
	}
	return expected == control
}

func companyFrom(f found, fallbackCUI string) *Company {
	addr := f.AdresaSediuSocial
	locality, sector := normalizeLocality(addr.Localitate)
	street := normalizeStreet(addr.Strada)
	if street == "" {
		street = strings.TrimSpace(addr.Detalii)
	}

	c := &Company{
		FiscalCode:      firstNonEmpty(strings.TrimSpace(f.DateGenerale.CUI.String()), fallbackCUI),
		Name:            strings.TrimSpace(f.DateGenerale.Denumire),
		TradeRegisterNo: strings.TrimSpace(f.DateGenerale.NrRegCom),
		Country:         "Romania",
		County:          strings.TrimSpace(addr.Judet),
		Locality:        locality,
		Street:          street,
		Number:          strings.TrimSpace(addr.Numar),
		Sector:          sector,
	}
	if f.InregistrareScopTVA.ScpTVA {
		c.Attribute = "RO"
	}
	return c
}

// CleanFiscalCode strips the optional RO prefix and every non-digit character,
// yielding the canonical digits-only form used for comparisons and storage.
func CleanFiscalCode(value string) string {
	value = strings.TrimSpace(strings.ToUpper(value))
	value = strings.TrimPrefix(value, "RO")
	return nonDigitRE.ReplaceAllString(value, "")
}

func normalizeStreet(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "Str. ")
	value = strings.TrimPrefix(value, "Str.")
	return strings.TrimSpace(value)
}

func normalizeLocality(value string) (string, string) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "Ors. ")
	value = strings.TrimPrefix(value, "Ors.")
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "Sector ") {
		parts := strings.Fields(value)
		if len(parts) >= 2 {
			return strings.TrimSpace(strings.TrimPrefix(value, "Sector "+parts[1])), parts[1]
		}
	}
	return value, ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
