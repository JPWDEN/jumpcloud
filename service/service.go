package service

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jumpcloud/types"
)

//Server interface allows for generic/template-based methods
//We dont have polymorphic stuff in this project...But if we DID, it would be here!
type Server interface {
	HashPassword(resp http.ResponseWriter, req *http.Request)
	CheckPassword(resp http.ResponseWriter, req *http.Request)
	GetAPIStats(resp http.ResponseWriter, req *http.Request)
	Shutdown(resp http.ResponseWriter, req *http.Request)
}

//ServerType holds the member variables for the server.
type ServerType struct {
	//Head points to position of a new id on the next API request to /hash
	Head int
	//IDMap holds time access and password data for each ID
	IDMap map[int]types.IDData
	//Average is a running average (in ms) of the time required to process all incoming hash requests
	Average int64
}

//NewServer is a constructor that initializes a server object of type ServerType
func NewServer() *ServerType {
	newMap := make(map[int]types.IDData)
	return &ServerType{IDMap: newMap}
}

func encodeBody(resp http.ResponseWriter, req *http.Request, data interface{}) error {
	return json.NewEncoder(resp).Encode(data)
}

func decodeBody(req *http.Request, data interface{}) error {
	defer req.Body.Close()
	return json.NewDecoder(req.Body).Decode(data)
}

func respond(resp http.ResponseWriter, req *http.Request, status int, data interface{}) {
	resp.WriteHeader(status)
	if data != nil {
		encodeBody(resp, req, data)
	}
}

func respondErr(resp http.ResponseWriter, req *http.Request, status int, args ...interface{}) {
	respond(resp, req, status, map[string]interface{}{
		"error": map[string]interface{}{"message": fmt.Sprint(args...)},
	})
}

func respondHTTPErr(resp http.ResponseWriter, req *http.Request, status int) {
	respondErr(resp, req, status, http.StatusText(status))
}

//HashAndEncrypt performs a SHA512 hash on the password provided, encodes to Base64, and returns the result
func hashAndEncrypt(password string) string {
	hash512 := sha512.New()
	hash512.Write([]byte(password))
	encoded := base64.StdEncoding.EncodeToString(hash512.Sum(nil))
	fmt.Printf("Hashed and encoded password: %s\n", encoded)
	return string(encoded)
}

//HashPassword fulfills implementation for the /hash and /hash/ endpoints
func (svr *ServerType) HashPassword(resp http.ResponseWriter, req *http.Request) {
	now := time.Now()
	path := req.URL.Path
	pathArgs := strings.Split(strings.Trim(path, "/"), "/")

	m, _ := url.ParseQuery(req.URL.RawQuery)
	fmt.Printf("Path args: %+v, raw %s, m %+v\n", pathArgs, req.URL.RawQuery, m)
	req.ParseForm()
	switch req.Method {
	case "POST":
		value, ok := req.Form["password"]
		if ok {
			resp.Write([]byte(strconv.Itoa(svr.Head))) //Write out the ID immediately to an http response
			hashedPW := hashAndEncrypt(value[0])
			//fmt.Printf("Password: %s\n", hashedPW)
			svr.IDMap[svr.Head] = types.IDData{Password: hashedPW, FirstCall: time.Now()}
			svr.Head++
		} else {
			respondHTTPErr(resp, req, http.StatusBadRequest)
		}
		elapsed := time.Since(now)
		fmt.Printf("Elapsed: %d\n", elapsed)
		svr.Average = ((svr.Average + elapsed.Nanoseconds()) / int64(svr.Head)) / 1000
		return
	default:
		respondHTTPErr(resp, req, http.StatusBadRequest)
		return
	}
}

//CheckPassword makes sure the 5-second wait has expired for a given ID.  If so, it returns the password
func (svr *ServerType) CheckPassword(resp http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	pathArgs := strings.Split(strings.Trim(path, "/"), "/")

	m, _ := url.ParseQuery(req.URL.RawQuery)
	fmt.Printf("Path args: %+v, raw %s, m %+v\n", pathArgs, req.URL.RawQuery, m)
	switch req.Method {
	case "GET":
		fmt.Println("In GET")
		id, err := strconv.Atoi(pathArgs[1])
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		value, ok := svr.IDMap[id]
		if ok {
			now := time.Now()
			fiveSecAgo := now.Add(time.Second * -5)
			if value.FirstCall.After(fiveSecAgo) {
				fmt.Printf("ID: %s\n", strconv.Itoa(id))
				resp.Write([]byte(strconv.Itoa(id)))
			} else {
				fmt.Printf("ID: %s\n", value.Password)
				resp.Write([]byte(value.Password))
			}
		}
		return
	default:
		respondHTTPErr(resp, req, http.StatusBadRequest)
		return
	}
}

func (svr *ServerType) GetAPIStats(resp http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	pathArgs := strings.Split(strings.Trim(path, "/"), "/")

	m, _ := url.ParseQuery(req.URL.RawQuery)
	fmt.Printf("Path args: %+v, raw %s, m %+v\n", pathArgs, req.URL.RawQuery, m)
	switch req.Method {
	case "GET":
		stats := types.StatsData{Total: svr.Head, Average: svr.Average}
		respond(resp, req, http.StatusOK, &stats)
		return
	default:
		respondHTTPErr(resp, req, http.StatusBadRequest)
		return
	}
}

func (svr *ServerType) Shutdown(resp http.ResponseWriter, req *http.Request) {

}
