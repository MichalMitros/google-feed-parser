package commander

// ParseCommand is command sent to Parser service.
type ParseCommand struct {
	ShopURL string `json:"shopUrl"`
}
