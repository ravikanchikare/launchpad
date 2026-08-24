package gateway

type Settings struct {
	GatewayURL     string `json:"gatewayUrl"`
	DefaultKeySlug string `json:"defaultKeySlug,omitempty"`
	Token          string `json:"-"`
}

type Model struct {
	ID      string `json:"id"`
	OwnedBy string `json:"owned_by,omitempty"`
}

type Config interface {
	GatewayURL() string
	Token() string
}
