FROM golang:1.24-alpine AS build

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o service main.go

FROM alpine:edge
WORKDIR /app
COPY --from=build /app/service .
ENTRYPOINT ["/app/service"]