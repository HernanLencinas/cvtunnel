FROM golang:1.24.0-alpine
ENV GOPATH=/go
ENV GOMOD=/go/github.com/HernanLencinas/cvtunnel/go.mod
WORKDIR /go/github.com/HernanLencinas/cvtunnel
COPY . ./
RUN go mod download
RUN go build -o ./cvtun main.go
EXPOSE 8000
ENTRYPOINT ["cvtun"]