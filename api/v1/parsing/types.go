package parsing

type requestBody struct {
	URL      string `json:"url" binding:"required"`
	SendHTML bool   `json:"send_html"`
}
