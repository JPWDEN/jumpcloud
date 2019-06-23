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

//StatsData is a JSON object to provide data on total number of calls and average response times
//Keep Average as a flot to maintain precision.  GetAPIStats will return Average * 10e-6
type StatsData struct {
	Total   int   `json:"total"`
	Average int64 `json:"average"`
}
