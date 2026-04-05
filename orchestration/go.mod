module github.com/TresPies-source/AgenticGatewayByDojoGenesis/orchestration

go 1.25.6

require (
	github.com/TresPies-source/AgenticGatewayByDojoGenesis/disposition v0.0.0
	github.com/TresPies-source/AgenticGatewayByDojoGenesis/skill v0.0.0
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/TresPies-source/AgenticGatewayByDojoGenesis/disposition => ../disposition
	github.com/TresPies-source/AgenticGatewayByDojoGenesis/skill => ../skill
)
