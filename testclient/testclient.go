package testclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/jumpcloud/types"
)

//ClientType object provides user data for testing
type ClientType struct {
	passwordList []string
	infoLog      *log.Logger
	errorLog     *log.Logger
}

//NewClient is the constructor for the testclient object
func NewClient(passwords []string, infoLog *log.Logger, errorLog *log.Logger) (*ClientType, error) {
	if len(passwords) < 1 {
		return nil, fmt.Errorf("No passwords received")
	}
	return &ClientType{
		passwordList: passwords,
		infoLog:      infoLog,
		errorLog:     errorLog,
	}, nil
}

//Run the HashPassword service function with test data
//Form:  curl -v --data "password=angryMonkey" -X POST localhost:8080/hash
//JSON:  curl -v --data '{"password":"angryMonkey"}' -H "Content-type: application/json" localhost:8080/hash
func (client *ClientType) runHashPassword(useJSON bool) {
	route := "http://localhost:8080/hash"
	if useJSON {
		payload := types.HashData{Password: client.passwordList[0]}
		byteMap, err := json.Marshal(payload)
		if err != nil {
			client.errorLog.Printf("HashPassword Error: %v", err)
			return
		}
		resp, err := http.Post(route, "application/json", bytes.NewBuffer(byteMap))
		if err != nil {
			client.errorLog.Printf("HashPassword Error: %v", err)
			return
		}
		var result types.HashData
		json.NewDecoder(resp.Body).Decode(&result)
		client.infoLog.Printf("HashPassword result: %+v\n", result)
	} else {
		payload := url.Values{}
		payload.Set("password", client.passwordList[0])
		resp, err := http.PostForm(route, payload)
		if err != nil {
			client.errorLog.Printf("HashPassword Error: %v", err)
			return
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		client.infoLog.Printf("HashPassword result: %+v\n", string(body))
	}
}

//Run the CheckPassword service function with test data
//curl -v localhost:8080/hash/{id}
func (client *ClientType) runCheckPassword(id int) {
	route := fmt.Sprintf("http://localhost:8080/hash/%d", id)
	req, err := http.NewRequest("GET", route, nil)
	if err != nil {
		client.errorLog.Printf("CheckPassword Error: %v", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		client.errorLog.Printf("CheckPassword Error: %v", err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	client.infoLog.Printf("CheckPassword result: %v\n", string(body))
}

//Run the GetAPIStats service function to check test data in previous calls
//curl -v localhost:8080/stats
func (client *ClientType) runGetAPIStats() {
	route := "http://localhost:8080/stats"
	req, err := http.NewRequest("GET", route, nil)
	if err != nil {
		client.errorLog.Printf("GetAPIStats Error: %v", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		client.errorLog.Printf("GetAPIStats Error: %v", err)
		return
	}
	defer resp.Body.Close()
	var result types.StatsData
	json.NewDecoder(resp.Body).Decode(&result)
	client.infoLog.Printf("GetAPIStats result: %+v\n", result)
}

//Run the Shutdown service function to test its affect on the service and other calls
//curl -v localhost:8080/shutdown
func (client *ClientType) runShutdown() {
	route := "http://localhost:8080/shutdown"
	req, err := http.NewRequest("GET", route, nil)
	if err != nil {
		client.errorLog.Printf("Shutdown Error: %v", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		client.errorLog.Printf("Shutdown Error: %v", err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	client.infoLog.Printf("Shutdown result: %v\n", string(body))
}

//RunClient does some simple tests on the API calls for validation on test data
func (client *ClientType) RunClient() {
	doneCH := make(chan bool)
	useJSON := false
	go func() {
		client.runHashPassword(useJSON) //1 of 6
		client.runHashPassword(useJSON) //2 of 6
		client.runCheckPassword(1)
		time.Sleep(time.Second * 5)
		client.runHashPassword(useJSON) //3 of 6
		client.runCheckPassword(1)
		client.runCheckPassword(2)
		client.runCheckPassword(3)
	}()
	go func() {
		client.runHashPassword(useJSON) //4 of 6
		client.runCheckPassword(1)
		client.runHashPassword(useJSON) //5 of 6
		client.runHashPassword(useJSON) // 6 of 6
		time.Sleep(time.Second * 5)
		client.runCheckPassword(4)
		client.runCheckPassword(5)
		client.runCheckPassword(6)

		doneCH <- true
	}()

	<-doneCH

	client.runGetAPIStats()
	client.runShutdown()

	//Below calls should not produce any meaningful output
	client.runHashPassword(useJSON)
	client.runCheckPassword(2)
	client.runGetAPIStats()
}
