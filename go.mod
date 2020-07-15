module gitlab.globoi.com/tks/gks/gks-operator

go 1.13

require (
	github.com/cosiner/argv v0.1.0 // indirect
	github.com/go-delve/delve v1.4.1 // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/operator-framework/operator-sdk v0.17.0
	github.com/peterh/liner v1.2.0 // indirect
	github.com/spf13/cobra v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5
	go.starlark.net v0.0.0-20200707032745-474f21a9602d // indirect
	golang.org/x/arch v0.0.0-20200511175325-f7c78586839d // indirect
	gopkg.in/yaml.v2 v2.3.0 // indirect
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
)
