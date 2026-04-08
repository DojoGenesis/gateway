module github.com/DojoGenesis/gateway/channel/slack

go 1.25.6

require (
	github.com/DojoGenesis/gateway/channel v0.0.0
	github.com/google/uuid v1.6.0
	github.com/slack-go/slack v0.15.0
)

require (
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
)

replace github.com/DojoGenesis/gateway/channel => ../
