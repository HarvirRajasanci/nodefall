FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/auth ./services/auth

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/auth /bin/auth
EXPOSE 8082
ENTRYPOINT ["/bin/auth"]
