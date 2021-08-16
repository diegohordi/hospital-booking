FROM golang:1.16.7-alpine3.14 as test
ENV GOOS linux
ENV CGO_ENABLED 0
RUN mkdir /app
COPY . /app
WORKDIR /app
RUN go mod download
CMD go test -count=1 -v ./...


FROM golang:1.16.7-alpine3.14 as build
ENV GOOS linux
ENV CGO_ENABLED 0
RUN mkdir /app
COPY . /app
WORKDIR /app
RUN go mod download
RUN go build -o restapi cmd/restapi/main.go


FROM alpine:3.14 as deploy
ARG SERVER_PORT
ARG DATABASE_DSN
ARG DATABASE_DRIVER
ARG PRIVATE_KEY_FILE_PATH
ARG PRIVATE_KEY_FILE

ENV PRIVATE_KEY_FILE=$PRIVATE_KEY_FILE
ENV SERVER_PORT=$SERVER_PORT
ENV DATABASE_DSN=$DATABASE_DSN
ENV DATABASE_DRIVER=$DATABASE_DRIVER

RUN apk add --no-cache curl
RUN apk add --no-cache ca-certificates
COPY --from=build /app/restapi /app/restapi
COPY $PRIVATE_KEY_FILE_PATH /app/$PRIVATE_KEY_FILE
EXPOSE $SERVER_PORT
CMD cd /app/ && ./restapi