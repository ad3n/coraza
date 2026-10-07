module github.com/ad3n/coraza/v3

go 1.27

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
	github.com/ad3n/jsonschema v0.0.7
	github.com/corazawaf/coraza-coreruleset v0.0.0-20240226094324-415b1017abdc
	github.com/corazawaf/libinjection-go v0.3.3
	github.com/foxcpp/go-mockdns v1.2.0
	github.com/jcchavezs/mergefs v0.1.1
	github.com/magefile/mage v1.17.2
	github.com/mccutchen/go-httpbin/v2 v2.25.0
	github.com/petar-dambovaliev/aho-corasick v0.0.0-20250424160509-463d218d4745
	github.com/valllabh/ocsf-schema-golang v1.0.3
	golang.org/x/net v0.59.0
	golang.org/x/sync v0.23.0
	rsc.io/binaryregexp v0.2.0
)

require (
	github.com/goccy/go-yaml v1.19.2 // indirect
	github.com/kaptinlin/jsonpointer v0.4.28 // indirect
	github.com/miekg/dns v1.1.57 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.2 // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/tools v0.49.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

retract v3.2.2
