package service

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

//ServerType holds the member variables for the server, includex a mutex for concurency safety.
type ServerType struct {
	//RWMutex provides read and write mutexes for safe concurrent read/write operations
	mux sync.RWMutex
	//Head points to position of a new id on the next API request to /hash
	Head int
	//IDMap holds time access and password data for each ID
	IDMap map[int]types.IDData
	//Average is a running average (purposely stored here in nanoseconds) of time required to process all incoming hash requests
	Average int64
	//shutdown (unexported) holds the shutdown status of the service
	shutdown bool
	//Below logs provide info- and error-specific logging within the service
	infoLog  *log.Logger
	errorLog *log.Logger
}

//NewServer is a constructor that initializes a server object of type ServerType
func NewServer(infoLog *log.Logger, errorLog *log.Logger) *ServerType {
	newMap := make(map[int]types.IDData)
	return &ServerType{
		IDMap:    newMap,
		shutdown: false,
		infoLog:  infoLog,
		errorLog: errorLog,
	}
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

//HashAndEncode performs a SHA512 hash on password, encodes to Base64, and returns the result
func hashAndEncode(password string) string {
	hash512 := sha512.New()
	hash512.Write([]byte(password))
	encoded := base64.StdEncoding.EncodeToString(hash512.Sum(nil))
	return string(encoded)
}

//HashPassword fulfills implementation for the /hash and /hash/ endpoints
//Per instructions, these endpoints do not process JSON requests; this function includes POC for processing JSON requests also
func (svr *ServerType) HashPassword(resp http.ResponseWriter, req *http.Request) {
	now := time.Now() //Duration is customer experience.  Prioritize this metric over checking shutdown
	svr.mux.RLock()
	stop := svr.shutdown
	svr.mux.RUnlock()
	if stop {
		//Uncomment to provide a shutdown response
		//resp.Write([]byte(fmt.Sprintf("Shutting service down\n")))
		return
	}
	//pathArgs := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	//m, _ := url.ParseQuery(req.URL.RawQuery)
	//svr.infoLog.Printf("Path args: %+v, raw %s, m %+v\n", pathArgs, req.URL.RawQuery, m)
	var passwd types.HashData

	//If JSON header exists, process as JSON.  If not, parse form data.
	useJSON := false
	if req.Header.Get("Content-type") == "application/json" {
		err := decodeBody(req, &passwd)
		if err != nil {
			respondErr(resp, req, http.StatusBadRequest, " Failed to decode body: ", err)
			svr.errorLog.Printf("Error in HashPassword: %v", err)
			return
		}
		useJSON = true
	} else {
		req.ParseForm()
		value, ok := req.Form["password"]
		if ok {
			passwd.Password = value[0]
		} else {
			resp.Write([]byte(fmt.Sprintf("Bad request:  No password.  Use \"password=<value>\"\n")))
			svr.errorLog.Printf("Error in HashPassword: No password.  Use \"password=<value>\"")
			return
		}
	}

	switch req.Method {
	case "POST":
		svr.mux.Lock()
		svr.Head++
		svr.IDMap[svr.Head] = types.IDData{Password: passwd.Password, FirstCall: now}
		sendHead := svr.Head
		elapsed := time.Since(now)
		svr.Average = ((svr.Average + elapsed.Nanoseconds()) / int64(svr.Head))
		svr.mux.Unlock()

		//Wait for 5 seconds before calculating hash
		go func() {
			time.Sleep(time.Second * 5)
			hashedPW := hashAndEncode(passwd.Password)
			svr.mux.Lock()
			svr.IDMap[sendHead] = types.IDData{Password: hashedPW, FirstCall: now}
			svr.mux.Unlock()
		}()

		//Meanwhile, write out the ID to an http response after incrementing head position above
		if useJSON {
			respond(resp, req, http.StatusOK, &types.HashData{Password: passwd.Password, ID: sendHead})
		} else {
			resp.Write([]byte(fmt.Sprintf("%s\n", strconv.Itoa(sendHead))))
		}
		svr.infoLog.Printf("Response return for HashPassword: %v", types.HashData{Password: passwd.Password, ID: sendHead})
		return
	default:
		if useJSON {
			respondHTTPErr(resp, req, http.StatusBadRequest)
		} else {
			resp.Write([]byte(fmt.Sprintf("Bad request: No POST\n")))
		}
		svr.errorLog.Printf("Error in HashPassword: %v", types.HashData{Password: passwd.Password, ID: svr.Head})
		return
	}
}

//CheckPassword makes sure the 5-second wait has expired for a given ID.  If so, it returns the password hash
func (svr *ServerType) CheckPassword(resp http.ResponseWriter, req *http.Request) {
	svr.mux.RLock()
	stop := svr.shutdown
	svr.mux.RUnlock()
	if stop {
		//Uncomment to provide a shutdown response
		//resp.Write([]byte(fmt.Sprintf("Shutting service down\n")))
		return
	}
	pathArgs := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	//m, _ := url.ParseQuery(req.URL.RawQuery)
	//svr.infoLog.Printf("Path args: %+v, raw %s, m %+v\n", pathArgs, req.URL.RawQuery, m)

	switch req.Method {
	case "GET":
		id, err := strconv.Atoi(pathArgs[1])
		if err != nil {
			resp.Write([]byte(fmt.Sprintf("Error in request: %v\n", err)))
			svr.errorLog.Printf("Error in request: %v\n", err)
			return
		}
		svr.mux.RLock()
		value, ok := svr.IDMap[id]
		svr.mux.RUnlock()
		var respString string
		if ok {
			now := time.Now()
			fiveSecAgo := now.Add(time.Second * -5)
			if value.FirstCall.After(fiveSecAgo) {
				respString = fmt.Sprintf("%s\n", strconv.Itoa(id))
			} else {
				respString = fmt.Sprintf("%s\n", value.Password)
			}
		} else {
			respString = fmt.Sprintf("Bad Request:  ID %d not found\n", id)
		}
		resp.Write([]byte(respString))
		svr.infoLog.Printf("Response return for CheckPassword: %s", respString)
		return
	default:
		respondHTTPErr(resp, req, http.StatusBadRequest)
		svr.errorLog.Printf("Error in CheckPassword: No GET method found in call")
		return
	}
}

//GetAPIStats returns a JSON object with total number of requests and average response time statistics
func (svr *ServerType) GetAPIStats(resp http.ResponseWriter, req *http.Request) {
	svr.mux.RLock()
	stop := svr.shutdown
	svr.mux.RUnlock()
	if stop {
		//Uncomment to provide a shutdown response
		//resp.Write([]byte(fmt.Sprintf("Shutting service down\n")))
		return
	}
	//pathArgs := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	//m, _ := url.ParseQuery(req.URL.RawQuery)
	//svr.infoLog.Printf("Path args: %+v, raw %s, m %+v\n", pathArgs, req.URL.RawQuery, m)
	switch req.Method {
	case "GET":
		svr.mux.RLock()
		stats := types.StatsData{Total: svr.Head, Average: svr.Average / 1000}
		svr.mux.RUnlock()
		respond(resp, req, http.StatusOK, &stats)
		svr.infoLog.Printf("Response return for GetAPIStats: %+v", stats)
		return
	default:
		respondHTTPErr(resp, req, http.StatusBadRequest)
		svr.errorLog.Printf("Error in GetAPIStats: No GET method found in call")
		return
	}
}

//Shutdown blocks API calls from use and initiates a graceful shutdown of the API service
func (svr *ServerType) Shutdown(resp http.ResponseWriter, req *http.Request) {
	svr.mux.Lock()
	svr.shutdown = true
	svr.mux.Unlock()
	resp.Write([]byte(fmt.Sprintf("Shutting service down\n")))

	go func() {
		time.Sleep(time.Second * 5) //Sleep to allow services to complete
		err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		if err != nil {
			resp.Write([]byte(fmt.Sprintf("Error shutting down: %v", err)))
			svr.errorLog.Printf("Error in Shutdown: %v", err)
			//If we want to keep processing API calls in the event we cannot shutdown, uncomment below code
			//svr.status.mux.Lock()
			//svr.status.shutdown = false
			//svr.status.mux.Unlock()
		}
	}()
	return
}
