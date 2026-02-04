FROM ubuntu AS build
RUN apt-get -y update && apt-get -y install golang git

RUN git clone "https://github.com/Story-Of-Alicia/authentication-authorization-http-server.git" "/tmp/output"

WORKDIR "/tmp/output"
RUN go mod tidy
RUN go build "cmd/server.go"

FROM ubuntu
RUN apt-get -y update && apt-get -y install golang ca-certificates

WORKDIR "/opt/soaauth"
COPY --from=build "/tmp/output/server" "/opt/soaauth/server"
RUN chmod a+x "/opt/soaauth/server"

CMD ["/opt/soaauth/server"]