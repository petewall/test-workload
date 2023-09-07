FROM golang

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go config.go ./
RUN mkdir /config
COPY config.yaml /config/config.yaml

ENV CONFIG_FILE_PATH=/config/config.yaml
# RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /simple-webapp

EXPOSE 3000
CMD ["go", "run", "."]
