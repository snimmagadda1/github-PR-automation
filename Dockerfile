FROM golang
ENV GO111MODULE=on

ENV APP_NAME github-pr-bot
ENV PORT 3000

# App specific vars
ENV APP_ID 111
ENV OWNER test
# ENV GITHUB_ENTERPRISE_URL
# ENV GITHUB_ENTERPRISE_UPLOAD_URL
ENV CERT_PATH testpath
ENV RELEASE_BRANCH releasetest
ENV MASTER_BRANCH master
ENV REPOS test

RUN set -x && go version
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN mkdir keys
RUN go build -o main cmd/main.go
CMD ["/app/main"]
