# build command:
# docker build --rm -t gocat-server .

FROM golang:1.17.4

RUN mkdir /app

ADD . /app

WORKDIR /app/src

RUN go build -o main .

CMD ["/app/src/main"]

