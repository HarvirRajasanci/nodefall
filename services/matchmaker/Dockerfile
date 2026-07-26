FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/matchmaker ./services/matchmaker

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/matchmaker /bin/matchmaker
EXPOSE 8083
ENTRYPOINT ["/bin/matchmaker"]
