FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /app cmd/app/main.go

FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=build /app /app
COPY migrations /migrations
EXPOSE 8080
ENTRYPOINT ["/app"]
