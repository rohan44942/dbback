FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/controller ./cmd/controller
RUN CGO_ENABLED=0 go build -o /app/dbback ./cmd/dbback

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /app/controller /usr/local/bin/controller
COPY --from=build /app/dbback /usr/local/bin/dbback
RUN mkdir -p logs backups configs
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/controller"]
