package types

import (
	"time"
)

//IDData holds information about the password and the first time this ID called the API
type IDData struct {
	Password  string
	FirstCall time.Time
}

//HashData is a JSON object that correlates to a JSON input on hash POST calls
type HashData struct {
	Password string `json:"password"`
	ID       int    `json:"id"`
}

//StatsData is a JSON object for responses to provide data on total API calls and average response times
type StatsData struct {
	Total   int     `json:"total"`
	Average float64 `json:"average"`
}
