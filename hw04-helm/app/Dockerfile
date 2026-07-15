FROM --platform=linux/amd64 golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /user-service .

FROM scratch
COPY --from=build /user-service /user-service
EXPOSE 8000
ENTRYPOINT ["/user-service"]
