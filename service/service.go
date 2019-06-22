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
	"sync"
	"syscall"
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

type safeShutdown struct {
	shutdown bool
	mux      sync.Mutex
}

//ServerType holds the member variables for the server.
type ServerType struct {
	//Head points to position of a new id on the next API request to /hash
	Head int
	//IDMap holds time access and password data for each ID
	IDMap map[int]types.IDData
	//Average is a running average (in ms) of the time required to process all incoming hash requests
	Average float64
	//status (unexported) holds the shutdown status of the service, and a lock for that status
	status safeShutdown
}

//NewServer is a constructor that initializes a server object of type ServerType
func NewServer() *ServerType {
	newMap := make(map[int]types.IDData)
	return &ServerType{IDMap: newMap, status: safeShutdown{shutdown: false}}
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

//HashAndEncrypt performs a SHA512 hash on password, encodes to Base64, and returns the result
func hashAndEncrypt(password string) string {
	hash512 := sha512.New()
	hash512.Write([]byte(password))
	encoded := base64.StdEncoding.EncodeToString(hash512.Sum(nil))
	fmt.Printf("Hashed and encoded password: %s\n", encoded)
	return string(encoded)
}

//HashPassword fulfills implementation for the /hash and /hash/ endpoints
//Per instructions, these endpoints do not process JSON requests; this function includes a POC for also processing JSON requests
func (svr *ServerType) HashPassword(resp http.ResponseWriter, req *http.Request) {
	now := time.Now() //Duration is customer experience.  Prioritize this metric over checking shutdown
	svr.status.mux.Lock()
	if svr.status.shutdown {
		resp.Write([]byte(fmt.Sprintf("Shutting service down\n")))
		return
	}
	svr.status.mux.Unlock()
	path := req.URL.Path
	pathArgs := strings.Split(strings.Trim(path, "/"), "/")
	m, _ := url.ParseQuery(req.URL.RawQuery)
	fmt.Printf("Path args: %+v, raw %s, m %+v\n", pathArgs, req.URL.RawQuery, m)
	fmt.Printf("json header: %s\n", req.Header.Get("Content-type"))
	var passwd types.HashData

	//If JSON header exists, process for JSON.  If not, parse form data.
	useJSON := false
	if req.Header.Get("Content-type") == "application/json" {
		err := decodeBody(req, &passwd)
		if err != nil {
			respondErr(resp, req, http.StatusBadRequest, " Failed to decode body: ", err)
			return
		}
		useJSON = true
	} else {
		req.ParseForm()
		value, ok := req.Form["password"]
		if ok {
			passwd.Password = value[0]
		} else {
			respondHTTPErr(resp, req, http.StatusBadRequest)
			return
		}
	}

	switch req.Method {
	case "POST":
		svr.Head++
		//Write out the ID to an http response after incrementing head position
		if useJSON {
			respond(resp, req, http.StatusOK, &types.HashData{ID: svr.Head})
		} else {
			resp.Write([]byte(fmt.Sprintf("%s\n", strconv.Itoa(svr.Head))))
		}
		hashedPW := hashAndEncrypt(passwd.Password)
		svr.IDMap[svr.Head] = types.IDData{Password: hashedPW, FirstCall: time.Now()}
		elapsed := time.Since(now)
		fmt.Printf("Elapsed: %d\n", elapsed)
		svr.Average = ((svr.Average + float64(elapsed.Nanoseconds())) / float64(svr.Head))
		return
	default:
		respondHTTPErr(resp, req, http.StatusBadRequest)
		return
	}
}

//CheckPassword makes sure the 5-second wait has expired for a given ID.  If so, it returns the password hash
func (svr *ServerType) CheckPassword(resp http.ResponseWriter, req *http.Request) {
	if svr.status.shutdown {
		resp.Write([]byte(fmt.Sprintf("Shutting service down\n")))
		return
	}
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
				resp.Write([]byte(fmt.Sprintf("%s\n", strconv.Itoa(id))))
			} else {
				fmt.Printf("ID: %s\n", value.Password)
				resp.Write([]byte(fmt.Sprintf("%s\n", value.Password)))
			}
		}
		return
	default:
		respondHTTPErr(resp, req, http.StatusBadRequest)
		return
	}
}

//GetAPIStats returns a JSON object with total number of requests and average response time statistics
func (svr *ServerType) GetAPIStats(resp http.ResponseWriter, req *http.Request) {
	if svr.status.shutdown {
		resp.Write([]byte(fmt.Sprintf("Shutting service down\n")))
		return
	}
	path := req.URL.Path
	pathArgs := strings.Split(strings.Trim(path, "/"), "/")

	m, _ := url.ParseQuery(req.URL.RawQuery)
	fmt.Printf("Path args: %+v, raw %s, m %+v\n", pathArgs, req.URL.RawQuery, m)
	switch req.Method {
	case "GET":
		stats := types.StatsData{Total: svr.Head, Average: svr.Average / 1000000.0}
		respond(resp, req, http.StatusOK, &stats)
		return
	default:
		respondHTTPErr(resp, req, http.StatusBadRequest)
		return
	}
}

//Shutdown blocks API calls from use and initiates a graceful shutdown of the API service
func (svr *ServerType) Shutdown(resp http.ResponseWriter, req *http.Request) {
	svr.status.mux.Lock()
	svr.status.shutdown = true
	svr.status.mux.Unlock()
	resp.Write([]byte(fmt.Sprintf("Shutting service down\n")))

	go func() {
		time.Sleep(time.Second * 5) //Sleep to allow services to complete
		err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		if err != nil {
			resp.Write([]byte(fmt.Sprintf("Error shutting down: %v", err)))
			//If we want to keep processing API calls in the event we cannot shutdown, uncomment below code
			//svr.status.mux.Lock()
			//svr.status.shutdown = false
			//svr.status.mux.Unlock()
		}
	}()
	return
}
