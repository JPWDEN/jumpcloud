# jumpcloud
The jumpcloud repository is a simple API server over HTTP that provides SHA512 password hashing for the caller, statistics on number of hash calls and average response time, and a graceful shutdown mechanism.  The implementation is thread/concurrency-safe, uses only standard Golang packages, and while it was not requested, the repository also has a basic Dockerfile and docker-compose.yaml, making it suitable for integration into a Kubernetes cluster.  The following endpoints can be found in the service package:
- `/hash` accepts a plaintext password and responds to the caller with a unique ID number for their hash.  Directions for this assessment called for password data to be sent as a Form.  As such, this endpoint processes that type of data.  Additionally, this endpoint includes a proof of concept for processing JSON data, and therefore can process both:  Form data receives generic response data in return, while JSON requests will receive JSON responses with both ID as well as plaintext password.  JSON requests *must* be accompanied by a “Content-type: application/json” header.<br>
Form request: `curl -v --data "password=angryMonkey" -X POST localhost:8080/hash` <br>
Form Response:
`1`<br>
JSON request: `curl -v --data '{"password":"angryMonkey"}' -H "Content-type: application/json" localhost:8080/hash`<br>
JSON response:
`{"password":"angryMonkey","id":1}`


- `/hash/{ID}` responds to the caller with the ID number if it has been less than 5 seconds since the hash was initiated.  If more than 5 seconds have passed, the endpoint will respond with the SHA512 hashed password encoded in Base64.<br>
Request:  `curl -v localhost:8080/hash/2`<br>
Response with less than 5 seconds elapsed time since /hash was called:
`2`<br>
Response with 5 seconds or more of elapsed time since /has was called:
`ZEHhWB65gUlzdVwtDQArEyx+KVLzp/aTaRaPlBzYRIFj6vjFdqEb0Q5B8zVKCZ0vKbZPZklJz0Fd7su2A+gf7Q==`

- `/stats` takes no parameters and responds in JSON format with an object containing the total number of *successful* hash requests and the average response time of those requests.  Internally, Average is stored in nanooseconds to maintain precision.  This endpoint will return the Average after converting to milliseconds to reflect the requested units.<br>
Request:  `curl -v localhost:8080/stats`<br>
Response:
`{"total":2,"average":73}`

- `/shutdown` initiates a graceful shutdown of the server.  This includes blocking the processing of any subsequent calls to the other endpoints, and a 5-second grace period for all endpoints to finish responding.<br>
Request: `curl -v localhost:8080/shutdown`<br>
Response:
`Shutting service down`<br><br>

# Additional packages
Two additional packages were written to support the processing that takes place in the server

## Logs
The logs package is a very simple wrapper around the standard log package.  It allows easy differentiation between logging that is informational, debug data, warnings, and errors.  In main, service, and testclient, only the informational (infoLog) and error (errorLog) logs are used.

## Testclient
The testclient package serves as a simple test framework.  Because this project needed to be safe for multiple concurrent calls and provide time-variant data, a test framework of this type felt more useful for some basic “pen testing”.  Unit tests could still be written using Go’s built in test framework, but it may prove to provide a lower ROI for a small project of this kind.

Example curl commands can also be found in testclient<br><br>

# Using this repo
To use, first clone the repo (`git clone git@github.com:JPWDEN/jumpcloud`) to a location that your $GOPATH is familiar with.

## Stand-alone
To run, navigate to the jumpcloud directory that was cloned and run:
`go run main.go`

The application will run and begin listening for HTTP requests on localhost:8080.

Open a new terminal window and issue curl commands to localhost:8080 to use the API.  Executing the curl command ` curl -v --data "password=angryMonkey" -X POST localhost:8080/hash ` as listed above should provide a response similar to `1`.  Log output will be available in the window the application is running from.

## Docker
Jumpcloud has a minimal Docker configuration and can be run in a container.
`docker-compose up` will build and run the container.
`docker-compose up -d` will run the container in the background.

Once the container is up and running, the application can be used exactly the same as the stand-alone method above, pointing to localhost:8080.<br><br>

# Other Notes
1. This project was written on a VM running Ubuntu 18.04.  Due to nuances in functionality with Docker, Go, and cURL, some modifications may be necessary to produce identical results on Windows platforms.<br>


# Appreciation
Thank you for your time reviewing this repo.  I'll be looking forward to hearing feedback.
