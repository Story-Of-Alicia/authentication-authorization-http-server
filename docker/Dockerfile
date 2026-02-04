FROM ubuntu
RUN apt-get -y update && apt-get -y install golang wget

EXPOSE 8080/tcp

RUN mkdir -p "/tmp/output"
RUN wget -q "https://github.com/Story-Of-Alicia/authentication-authorization-http-server/archive/refs/tags/RC.tar.gz" -O release.tar.gz
RUN tar -xf "release.tar.gz" -C "/tmp/output" --strip-components=1

RUN cd "/tmp/output/cmd"
RUN go mod tidy
RUN go build server.go

RUN mkdir -p "/opt/soaauth/server"
RUN cp server "/opt/soaauth/server"
RUN chmod a+x "/opt/soaauth/server"

CMD ["/opt/soaauth/server"]
