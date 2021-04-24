FROM golang
ENV GO111MODULE=on

ENV APP_NAME bot
ENV PORT 3000
RUN set -x && go version
COPY . /go/src/${APP_NAME}
WORKDIR /go/src/${APP_NAME}
RUN mkdir keys

RUN go get ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ${APP_NAME}
EXPOSE ${PORT}
CMD ./${APP_NAME}

