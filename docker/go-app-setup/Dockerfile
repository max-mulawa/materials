FROM golang:1.23 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY pkg/ ./pkg/

RUN CGO_ENABLED=0 GOOS=linux go build -o ./api cmd/web/main.go

FROM gcr.io/distroless/static-debian12

COPY --from=build /src/api /app/api

ENTRYPOINT [ "/app/api" ]
