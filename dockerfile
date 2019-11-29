FROM golang:1.13.4
WORKDIR /go/src/go-dns
ADD . .
RUN go build -o dns .
CMD ["/go/src/go-dns/dns"]