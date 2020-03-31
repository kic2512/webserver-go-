FROM golang:1.14.1-buster
EXPOSE 8080
EXPOSE 80
RUN mkdir /app
ADD . /app/
WORKDIR /app
ENV GOPATH /app
RUN go build httpd.go
ENTRYPOINT ["/app/httpd", "-r", "/app/httptest/", "-c", "2", "-p", "80"]
