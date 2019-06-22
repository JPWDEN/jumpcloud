package testclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/jumpcloud/types"
)

//Run the HashPassword service function with test data
//Form:  curl -v --data "password=angryMonkey" -X POST localhost:8080/hash
//JSON:  curl -v --data '{"password":"angryMonkey"}' -X POST localhost:8080/hash
func runHashPassword(useJSON bool) {
	route := "http://localhost:8080/hash"
	if useJSON {
		payload := types.HashData{Password: "angryMonkey"}
		byteMap, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf("HashPassword Error: %v", err)
			return
		}
		resp, err := http.Post(route, "application/json", bytes.NewBuffer(byteMap))
		if err != nil {
			fmt.Printf("HashPassword Error: %v", err)
			return
		}
		var result types.HashData
		json.NewDecoder(resp.Body).Decode(&result)
		fmt.Printf("HashPassword result: %+v\n", result)
	} else {
		payload := url.Values{}
		payload.Set("password", "angryMonkey")
		resp, err := http.PostForm(route, payload)
		if err != nil {
			fmt.Printf("HashPassword Error: %v", err)
			return
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("HashPassword result: %+v\n", string(body))
	}
}

//Run the CheckPassword service function with test data
//curl -v localhost:8080/hash/{id}
func runCheckPassword(id int) {
	route := fmt.Sprintf("http://localhost:8080/hash/%d", id)
	req, err := http.NewRequest("GET", route, nil)
	if err != nil {
		fmt.Printf("CheckPassword Error: %v", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("CheckPassword Error: %v", err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("CheckPassword result: %v\n", string(body))
}

//Run the GetAPIStats service function to check test data in previous calls
//curl -v localhost:8080/stats
func runGetAPIStats() {
	route := "http://localhost:8080/stats"
	req, err := http.NewRequest("GET", route, nil)
	if err != nil {
		fmt.Printf("GetAPIStats Error: %v", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("GetAPIStats Error: %v", err)
		return
	}
	defer resp.Body.Close()
	var result types.StatsData
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Printf("GetAPIStats result: %+v\n", result)
}

//Run the Shutdown service function to test its affect on the service and other calls
func runShutdown() {
	route := "http://localhost:8080/shutdown"
	req, err := http.NewRequest("GET", route, nil)
	if err != nil {
		fmt.Printf("Shutdown Error: %v", err)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Shutdown Error: %v", err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Shutdown result: %v\n", string(body))
}

//RunClient does some simple tests on the API calls for validation on test data
func RunClient() {
	doneCH := make(chan bool)
	useJSON := false
	go func() {
		runHashPassword(useJSON) //1 of 6
		runHashPassword(useJSON) //2 of 6
		runCheckPassword(1)
		time.Sleep(time.Second * 5)
		runHashPassword(useJSON) //3 of 6
		runCheckPassword(1)
		runCheckPassword(2)
		runCheckPassword(3)
	}()
	go func() {
		runHashPassword(useJSON) //4 of 6
		runCheckPassword(1)
		runHashPassword(useJSON) //5 of 6
		runHashPassword(useJSON) // 6 of 6
		time.Sleep(time.Second * 5)
		runCheckPassword(4)
		runCheckPassword(5)
		runCheckPassword(6)

		doneCH <- true
	}()

	<-doneCH

	runGetAPIStats()
	runShutdown()

	//Below calls should not produce any meaningful output
	runHashPassword(useJSON)
	runCheckPassword(2)
	runGetAPIStats()
}
