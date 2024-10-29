module github.com/marsevilspirit/m_RPC

go 1.23.2

require github.com/fatih/color v1.18.0

require (
	github.com/kavu/go_reuseport v1.5.0
	github.com/stretchr/testify v1.9.0
	github.com/vmihailenco/msgpack/v5 v5.4.1
	google.golang.org/protobuf v1.35.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/coreos/bbolt => go.etcd.io/bbolt v1.3.11
