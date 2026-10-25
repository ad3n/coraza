module github.com/ad3n/coraza/v3

go 1.26.0

toolchain go1.27.3

// Testing dependencies:
// - go-mockdns
// - go-modsecurity (optional)

// Development dependencies:
// - mage

// Build dependencies:
// - libinjection-go
// - aho-corasick
// - gjson
// - binaryregexp
// - ocsf-schema-golang

require (
	github.com/ad3n/gjson v1.0.3
	github.com/ad3n/libinjection-go v1.0.0
	github.com/corazawaf/coraza-coreruleset/crs/tests/v4 v4.30.0
	github.com/corazawaf/coraza-coreruleset/crs/v4 v4.30.0
	github.com/foxcpp/go-mockdns v1.2.0
	github.com/jcchavezs/mergefs v0.1.1
	github.com/kaptinlin/jsonschema v0.4.6
	github.com/magefile/mage v1.17.2
	github.com/mccutchen/go-httpbin/v2 v2.25.0
	github.com/petar-dambovaliev/aho-corasick v0.0.0-20250424160509-463d218d4745
	github.com/valllabh/ocsf-schema-golang v1.0.3
	golang.org/x/net v0.60.0
	golang.org/x/sync v0.23.0
	rsc.io/binaryregexp v0.2.0
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/gotnospirit/makeplural v0.0.0-20180622080156-a5f48d94d976 // indirect
	github.com/gotnospirit/messageformat v0.0.0-20221001023931-dfe49f1eb092 // indirect
	github.com/kaptinlin/go-i18n v0.1.4 // indirect
	github.com/miekg/dns v1.1.57 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.2 // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	golang.org/x/tools v0.49.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

retract v3.2.2
