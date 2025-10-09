FROM golang:1.20-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /app/bin/dbback ./cmd/dbback

FROM alpine:3.18
COPY --from=build /app/bin/dbback /usr/local/bin/dbback
ENTRYPOINT ["/usr/local/bin/dbback"]
