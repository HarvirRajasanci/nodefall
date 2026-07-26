FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/game ./services/game

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/game /bin/game
EXPOSE 8081 9090
ENTRYPOINT ["/bin/game"]
