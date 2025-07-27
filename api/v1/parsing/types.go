package parsing

type requestBody struct {
	URL string `json:"url" binding:"required"`
}
