FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app .

FROM scratch
COPY --from=build /app /app
EXPOSE 8000
ENTRYPOINT ["/app"]
