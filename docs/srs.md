# Alexandria - Software Requirement Specification

## Overview

Alexandria is a book and resource server. It's goal is to take in epub, pdf and external websites and allow one to read them remotely. 

Key Features:

1. Open a book and be able to read via scrolling
2. Supports TTS which can easily be turned on, paused, started on a different paragraph, etc. (for epub)
3. Can categorize books, search them, and index them via common book attributes (author, genre, etc.) supported by epub metadata.
4. Runs via a server you can connect to, protected by auth

## Business Need

I want to read 50 books this year, and this project must support this.

## Technical Requirements

### Architectural Requirements

- Server must be developed in Go
    - Can use gin, fiber, or whichever is easiest
    - Must be built with this architecture:
        - Domain - Data structures, including DTOs
        - App - Interface layer for things
            - App - State and ties everything together
            - Ports - Capabilities 
            - Repos - Access to database, generic
            - Apis - How people access the app (Http Server, etc.)
            - Service - Business logic which calls these interfaces (repos and ports)
        - Infra - Individual implementations of ports and repos
        - Api - Implementations of APIs. Should only call services.
        - Config
- Cmd - Actual binary main.go's or typescript entrypoints themselves
    - Server
    - Web -  Web app in TypeScript
- Client - A vite webclient in VueJS or Vanilla TypeScript which accesses this
- Test - Must support e2e tests which can be easily spun up. My recommendation in Pytest (local) and docker compose, though go is another nice alternative
- Scripts - docker compose and other helpful scripts.

### Other

- Must support plain auth, passwords and accounts and such
- Must be secure to deploy publicly without people accessing books without login
- Try to have a nice interface
- Entire repo must be well tested (unit, integration, e2e) for rapid validation of features, try TDD for development
- Entire repo must be well documented, and extremely easy to spin up things (prefer writing scripts for 1 command operations)
- Deployable easily via docker.