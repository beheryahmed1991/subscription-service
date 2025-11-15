# syntax=docker/dockerfile:1.7

FROM golang:1.25 AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o subscription-service ./main.go

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /app/subscription-service /app/subscription-service
COPY --from=build /app/docs /app/docs

ENV PORT=8080
EXPOSE 8080

CMD ["/app/subscription-service"]
