FROM golang:1.24.2 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 go build -o bin/hue2govee github.com/cedrickring/hue-to-govee/cmd/hue2govee

FROM alpine

WORKDIR /app
USER 10001:10001

COPY --from=build /app/bin/hue2govee /app/hue2govee

CMD ["/app/hue2govee"]
