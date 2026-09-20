FROM golang:1.23-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o service ./cmd/app/main.go

FROM alpine:edge
WORKDIR /app
COPY --from=build /app/service .
ENTRYPOINT ["/app/service"]