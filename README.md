# Greenlight

Project built while following along with [Let's Go Further](https://lets-go-further.alexedwards.net/)
by [Alex Edwards](https://www.alexedwards.net/).

## Project Layout

### Bin

Contains compiled application binaries, ready for deployment to a production server.

### Cmd

Contains application specific code.
This will include the code for running the server, reading and writing HTTP requests, and managing authentication.

### Internal
Contains various ancillary packages used by our API.
Contains code for interacting with the database, doing data validation, sending emails, etc.

### Migrations
Contains the SQL migrations files for our database.

### Remote
Contains the configurations files and setup scripts for the production server.
