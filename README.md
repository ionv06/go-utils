# go-utils

Go utils library.

```
go get github.com/ionv06/go-utils
```

## Contents

**[Package `utils`](#package-utils)**

| Function | What it does |
|---|---|
| [`Clean(dir string) string`](#cleandir-string-string) | Normalizes a path: strips quotes, cleans it, drops trailing separators |
| [`LoadEnv() error`](#loadenv-error) | Finds a `.env` file and loads its keys into the process environment |

**[Package `anaf`](#package-anaf)**

| Function | What it does |
|---|---|
| [`Lookup(fiscalCode string) ([]byte, error)`](#lookupfiscalcode-string-byte-error) | Validates a CUI, downloads the company from ANAF, returns it as JSON |
| [`LookupCompany(fiscalCode string) (*Company, error)`](#lookupcompanyfiscalcode-string-company-error) | Same as `Lookup`, but returns the struct instead of JSON |
| [`CleanFiscalCode(value string) string`](#cleanfiscalcodevalue-string-string) | Strips the `RO` prefix and every non-digit, giving the canonical form |
| [`UserMessage(err error) (string, bool)`](#usermessageerr-error-string-bool) | Returns the user-safe text of an error, if it has one |

See also [Types and sentinel errors](#types-and-sentinel-errors) for `Company`,
`UserError` and the `Err*` values.

**[Tool `genversioninfo`](#tool-genversioninfo)** — generates `versioninfo.json`
from git metadata.

## Package `utils`

Small helpers for paths and environment.

```go
import "github.com/ionv06/go-utils/utils"
```

### `Clean(dir string) string`

Normalizes a filesystem path: strips surrounding double quotes, applies
`filepath.Clean`, and removes trailing `\` or `/` separators. Useful for paths
that arrive from command-line arguments or config files where the user may have
quoted them or left a trailing slash.

```go
utils.Clean(`"C:\data\exports\"`) // -> C:\data\exports
```

### `LoadEnv() error`

Finds a `.env` file and loads its keys into the process environment.

Search order (first match wins):

1. `.env`
2. `../.env`
3. `../../.env`
4. `.env` next to the running executable

File format: one `KEY=VALUE` per line. Blank lines and lines starting with `#`
are ignored. Surrounding double quotes are stripped from the value. Lines
without `=` are skipped.

Variables already present in the environment are **not** overwritten — real
environment variables always win over the `.env` file.

Returns `os.ErrNotExist` if no `.env` file was found in any location.

## Package `anaf`

Downloads Romanian company data from the ANAF public VAT registry web service
(`webservicesp.anaf.ro`, API v9). Each call is bounded by a 20s timeout.

```go
import "github.com/ionv06/go-utils/anaf"
```

### `Lookup(fiscalCode string) ([]byte, error)`

Validates the Romanian fiscal code (CUI), downloads the company data from ANAF
and returns it JSON-encoded (the encoding of `Company`). Callers must **not**
pre-validate the code — `Lookup` owns that step.

Accepts the code in any common form: `RO12345678`, `12345678`, with spaces or
punctuation.

### `LookupCompany(fiscalCode string) (*Company, error)`

Same as `Lookup` but returns the `*Company` struct directly, without the JSON
encoding step.

### `CleanFiscalCode(value string) string`

Strips the optional `RO` prefix and every non-digit character, yielding the
canonical digits-only form used for comparisons and storage.

```go
anaf.CleanFiscalCode(" ro 12.345.678 ") // -> 12345678
```

### `UserMessage(err error) (string, bool)`

Returns the user-safe text of an error, if it has one. Use it to decide what may
be shown to an end user: input problems and unknown companies are safe, while
network, TLS and decode failures are internal faults and must be logged instead.

```go
data, err := anaf.Lookup(code)
if err != nil {
    if msg, ok := anaf.UserMessage(err); ok {
        http.Error(w, msg, http.StatusBadRequest) // safe to show
        return
    }
    log.Println(err)                              // internal fault
    http.Error(w, "eroare interna", http.StatusInternalServerError)
    return
}
```

### Types and sentinel errors

`Company` is the normalized subset of the ANAF record:

| Field | JSON key | Notes |
|---|---|---|
| `FiscalCode` | `fiscal_code` | digits only |
| `Name` | `name` | |
| `TradeRegisterNo` | `trade_register_no` | |
| `Country` | `country` | always `Romania` |
| `County` | `county` | |
| `Locality` | `locality` | `Ors.` prefix and `Sector N` removed |
| `Street` | `street` | `Str.` prefix removed; falls back to address details |
| `Number` | `number` | |
| `Sector` | `sector,omitempty` | Bucharest sector number, when present |
| `Attribute` | `attribute,omitempty` | `RO` when the company is VAT registered |

`UserError` marks an error whose text is safe to show to the end user. The
sentinel values:

| Error | Message | Meaning |
|---|---|---|
| `ErrMissingCode` | `cod fiscal lipsa` | no fiscal code supplied |
| `ErrInvalidCode` | `cod fiscal invalid` | not a well-formed Romanian CUI |
| `ErrNotFound` | `cod fiscal negasit la ANAF` | ANAF knows no company with this code |

Validation requires 2–10 digits with a correct control digit (the official CUI
checksum).

## Tool `genversioninfo`

Generates a `versioninfo.json` in the current directory from git metadata, for
embedding version information into Windows executables.

```
go run github.com/ionv06/go-utils/tools/genversioninfo [flags]
```

The version comes from `git describe --tags --abbrev=0` (falling back to
`v0.0.0`) and is split into major/minor/patch. The short commit hash from
`git rev-parse --short HEAD` (falling back to `abcd`) is written to the
`Comments` field.

| Flag | Default |
|---|---|
| `-CompanyName` | `Cometa SRL` |
| `-FileDescription` | `ContabSQL Tools` |
| `-InternalName` | `contab_tools` |
| `-OriginalFilename` | `saft-xml2xlsx.exe` |
| `-ProductName` | `ERP Tools` |
