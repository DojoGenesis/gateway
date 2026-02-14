package handlers

type SSEBroadcasterImpl struct{}

func NewSSEBroadcaster() *SSEBroadcasterImpl {
	return &SSEBroadcasterImpl{}
}

func (b *SSEBroadcasterImpl) SendToClient(clientID string, event string, data string) error {
	return SendToClient(clientID, event, data)
}

func (b *SSEBroadcasterImpl) GetClientIDs() []string {
	return GetClientIDs()
}
