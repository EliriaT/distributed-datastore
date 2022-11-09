
# first stage - builds the binary from sources
FROM golang:alpine as build

# using build as current directory
WORKDIR /app

# adding the source code to current dir:
COPY . .

# downloading dependencies and verifying
RUN go mod download && go mod verify


# building the project
RUN go build  -o main .

# second stage - using minimal image to run the server
FROM alpine:latest

# using /app as current directory
WORKDIR /app

# copy server binary from `build` layer
COPY --from=build /app/main .


## copy a script for container sinchronization
#COPY wait-for.sh .

# binary to run
CMD "/app/main"

EXPOSE 8082

