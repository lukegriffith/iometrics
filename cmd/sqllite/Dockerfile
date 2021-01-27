FROM golang:1.7.3
WORKDIR /root/
RUN go get -d -v github.com/lukemgriffith/iometrics/cmd/sqllite
COPY main.go .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build  -o app .

FROM ubuntu:latest  
WORKDIR /root/
COPY --from=0 /root/app .
CMD ["/root/app"]  
