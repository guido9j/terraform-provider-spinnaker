module github.com/guido9j/terraform-provider-spinnaker

go 1.14

require (
	github.com/antihax/optional v1.0.0
	github.com/ghodss/yaml v1.0.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.23.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0
	github.com/spinnaker/spin v1.27.1
	google.golang.org/grpc v1.48.0 // indirect
)

replace git.apache.org/thrift.git => github.com/apache/thrift v0.0.0-20180902110319-2566ecd5d999
