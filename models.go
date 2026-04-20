package webostv

type Application struct {
	ID    string                 `json:"id"`
	Title string                 `json:"title"`
	Icon  string                 `json:"icon"`
	Data  map[string]interface{} `json:"-"`
}

type InputSource struct {
	ID    string                 `json:"id"`
	Label string                 `json:"label"`
	Data  map[string]interface{} `json:"-"`
}

type AudioOutputSource struct {
	Source string
}

func (a Application) String() string {
	return "<Application '" + a.Title + "'>"
}

func (i InputSource) String() string {
	return "<InputSource '" + i.Label + "'>"
}

func (a AudioOutputSource) String() string {
	return "<AudioOutputSource '" + a.Source + "'>"
}
