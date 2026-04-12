# Adds 10 singers to the queue with varied names and song selections.
# Run the app without rate limiting: make run-noenforce

POST http://localhost:8080/signup
[FormParams]
name: Jamie
song: 32
song: 9
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Priya
song: 15
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Marcus
song: 20
song: 25
song: 24
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Sofia
song: 22
song: 27
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Derek
song: 30
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Nora
song: 1
song: 35
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Eli
song: 33
song: 37
song: 38
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Carmen
song: 26
song: 39
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Theo
song: 19
HTTP 200

POST http://localhost:8080/signup
[FormParams]
name: Leila
song: 5
song: 31
HTTP 200