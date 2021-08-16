# Hospital Booking System

## Running

`make run` to run the app and `make stop` to stop it.

## API Design

There is an Open API v3 spec file  under /api directory, which you can import into your favorite test tool 
and play.

Notice that:

* `GET/POST {{baseUrl}}/api/v1/calendar/:doctorUUID/:year/:month/:day`, is restricted for the users with PATIENT role, allows 
patients to get a doctor's calendar or insert a new appointment into it.

Doctor UUID, e.g : 293691a7-9d90-47f9-a502-ff196f9d50e0


* GET `{{baseUrl}}/api/v1/calendar/:year/:month/:day`, is restricted for the users with DOCTOR role, allows
  doctors to get his/her own calendar with appointment details (if there are one).


* INSERT `{{baseUrl}}/api/v1/calendar/blockers`, is restricted for the users with DOCTOR role, allows
  doctors to insert a new block period into his/her calendar.

## Security

I implemented a signed JWT schema in order to exchange tokens. Furthermore, I created two middlewares, one
to check the JWT validity and one another to check the user's roles, both used to allow or not some requests. So, 
protected endpoints expects a valid Authorization header with "Bearer " + JWT Access Token.

The tokens are not stored into database and the default timeouts for access token is 10 minutes, and the refresh 
token 24 hours.

I also created a tool to generate key pairs, used to sign the generated JWT, which you can see the usage details
further.

* To login as a patient, use the following credentials:<br/>
  `{"email": "patient@hospital.com", "password": "patient"}`


* To login as a doctor, use the following credentials:<br/>
  `{"email": "doctor@hospital.com", "password": "doctor"}`

## Database

The system uses PostgreSQL as database and its template is in /deployments/database folder.

First, I decided to use PostgreSQL due to my familiarity, and second, an SQL database due to the project's
schema rigidity and the ACID characteristics - IMHO required for a booking system.

I didn't use any migration tool to create and seed the database in order to keep the things as simple
as possible, without any really needed external dependencies, but in production grade environments
I'm used to putting it in place.

I've used UUID strategy to expose row identifiers to the end users, but to keep things simple,
I didn't implement a collision check, but, of course, in production grade software
we must handle this properly.

## Tests

I achieved an average of 80% of code coverage. I didn't implement BDD, but I did the integration tests
covering all app layers, mocking the database. To run the tests, execute: <br/>

`make run_test`

## Architecture/Configuration

### Database
This image is based on the latest PostgreSQL image and uses a bash file to create the user, database
and also runs the hospital_booking.sql in order to create and seed the used database. It receives as 
parameters:
* APP_USER: The database username that should be used to access the database.
* APP_PASSWORD: The user's pass.
* APP_DB: The database name.

### Backend
This is a multi-stage image, one for the build stage that uses the golang:1.16.7-alpine3.14 image as basis,
and the other one, used for deployment, uses alpine:3.14 as basis. It receives as environment vars:
* DATABASE_DSN: Database DSN.
* DATABASE_DRIVER: Database driver.
* PRIVATE_KEY_FILE: Private key's file name.
* SERVER_PORT: Server port that should be exposed.

### Proxy
To avoid exposing the identity of the backend server, I put an NGINX as a reverse proxy. If no configuration
has been changed, the API should be accessible from `http://localhost/`

### Metrics, Logging and Monitoring
Uses E(lastic Search) L(ostash) K(ibana) stack. Logs are sent from backend to logstash by gelf logging driver.

The index that must be configured in Kibana is `backend-*` and if no configuration has been changed, 
Kibana will run on 5601 port, and then it's possible to access it from `http://localhost:5601/app/kibana`

For metrics, I've used Prometheus. If no configuration has been changed, it will run on 9090, and then it's possible
to access it from `http://localhost:9090`. The following metrics are in place:

* http_requests_total - Counts all requests by path
* http_duration - Duration of requests by path

## Tools

### passgen

Generates encrypted passwords, used to seed the database. <br/> 
`make passgen pass=mypass`

### uuidgen

Generates random UUID, used to seed the database. <br/>
`make uuidgen`

### keygen

Generates private and public keys used to sign JWT tokens. <br/>
`make keygen dir=configs`

