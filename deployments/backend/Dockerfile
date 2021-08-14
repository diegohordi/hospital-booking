FROM golang:1.16.7-alpine3.14 as builder
ENV GOOS linux
ENV CGO_ENABLED 0
RUN mkdir /app
COPY . /app
WORKDIR /app
RUN go mod download
RUN go build -o restapi cmd/restapi/main.go

FROM alpine:3.14 as production
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/restapi /app/restapi
COPY --from=builder /app/configs/config_prod.json /app/.
COPY --from=builder /app/configs/private.pem /app/.
# Expose port
EXPOSE 8080
# Exec built binary
CMD cd /app/ && ./restapi -config config_prod.json