FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/mock-psp ./cmd/mock-psp

FROM alpine:3.21 AS server
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/server /app/server
COPY migrations /app/migrations
ENV MIGRATIONS_PATH=/app/migrations
EXPOSE 8080
CMD ["/app/server"]

FROM alpine:3.21 AS mock-psp
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/mock-psp /app/mock-psp
EXPOSE 8081
CMD ["/app/mock-psp"]
