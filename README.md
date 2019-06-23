# jumpcloud
The jumpcloud repository is a simple API server over HTTP that provides SHA512 password hashing for the caller, statistics on number of hash calls and average response time, and a graceful shutdown mechanism.  The implementation is thread/concurrency-safe, and while it was not requested, the repository also has a basic Dockerfile and docker-compose.yaml, making it suitable for integration into a Kubernetes cluster.  The following endpoints can be found in the service package:
- `/hash` accepts a plaintext password and responds to the caller with a unique ID number for their hash.  Directions for this assessment called for password data to be sent as a Form.  As such, this endpoint processes that type of data.  Additionally, this endpoint includes a proof of concept for processing JSON data, and therefore can process both:  Form data receives generic response data in return, while JSON requests will receive JSON responses with both ID as well as plaintext password.  JSON requests *must* be accompanied by a “Content-type: application/json” header.

Form request: `curl -v --data "password=angryMonkey" -X POST localhost:8080/hash`
Form Response:
`1`
  
JSON request: `curl --data '{"password":"angryMonkey"}' -H "Content-type: application/json" localhost:8080/hash`
JSON response:
`{"password":"angryMonkey","id":1}`


- `/hash/{ID}` responds to the caller with the ID number if it has been less than 5 seconds since the hash was initiated.  If more than 5 seconds have passed, the endpoint will respond with the SHA512 hashed password encoded in Base64.

Request:  curl -v localhost:8080/hash/2
Response with less than 5 seconds elapsed time since /hash was called:
2

Response with 5 seconds or more of elapsed time since /has was called:
ZEHhWB65gUlzdVwtDQArEyx+KVLzp/aTaRaPlBzYRIFj6vjFdqEb0Q5B8zVKCZ0vKbZPZklJz0Fd7su2A+gf7Q==

- /stats takes no parameters and responds in JSON format with an object containing the total number of *successful* hash requests and the average response time of those requests.  Average is measured in microseconds.

Request:  curl localhost:8080/stats
Response: {"total":2,"average":73}

- /shutdown initiates a graceful shutdown of the server.  This includes blocking the processing of any subsequent calls to the other endpoints, and a 5-second grace period for all endpoints to finish responding.

Request: curl localhost:8080/shutdown
Response:
Shutting service down

# Additional packages
Two additional packages were written to support the processing that takes place in the server

## Logs
The logs package is a very simple wrapper around the standard log package.  It allows easy differentiation between logging that is informational, debug data, warnings, and errors.  In main, service, and testclient, only the informational (infoLog) and error (errorLog) logs are used.

## Testclient
The testclient package serves as a simple test framework.  Because this project needed to be safe for multiple concurrent calls and provide time-variant data, a test framework of this type felt more useful for some basic “pen testing”.  Unit tests could still be written using Go’s built in test framework, but it may prove to provide a lower ROI for a small project of this kind.

Example curl commands can also be found in testclient
