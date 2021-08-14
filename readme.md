# Hospital Booking System

#### WARNING
* I didn't deploy any production grade software using GO yet, so, any reference to this here refers to my prior 
experiences.

## Database

The system uses PostgreSQL as database and its template is in /deployments/database folder.

First, I decided to use PostgreSQL due to my familiarity, and second, an SQL database due to the project's
schema rigidity and the ACID characteristics, IMHO required for a booking system. 

I didn't use any migration tool to create and seed the database in order to keep the things as simple
as possible, without any really needed external dependencies, but in production grade environments 
I'm used to putting it in place.

I've used UUID strategy to expose row identifiers to the end users, but to keep things simple,
I didn't implement a collision check, but, of course, in production grade software
we must handle this properly.

## Tests

I achieved an average of 80% of code coverage. I didn't implement BDD, but I did the integration tests
covering all app layers.

## Security

I implemented a signed JWT schema in order to exchange tokens. Furthermore, I created two middlewares, one
to check the JWT validity and one another to check the user's, both used to allow or not some requests.

I also created a tool to generate key pairs, used to sign the generated JWT.

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

