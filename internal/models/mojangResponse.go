package models

type MojangResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Player struct {
			Meta struct {
				CachedAt int `json:"cached_at"`
			} `json:"meta"`
			Username    string `json:"username"`
			ID          string `json:"id"`
			RawID       string `json:"raw_id"`
			Avatar      string `json:"avatar"`
			SkinTexture string `json:"skin_texture"`
			Properties  []struct {
				Name      string `json:"name"`
				Value     string `json:"value"`
				Signature string `json:"signature"`
			} `json:"properties"`
			NameHistory []interface{} `json:"name_history"`
		} `json:"player"`
	} `json:"data"`
	Success bool `json:"success"`
}
