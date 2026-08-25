FROM docker.m.daocloud.io/library/golang:1.26.3-bookworm

ENV CGO_ENABLED=0
ENV GOTOOLCHAIN=local
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/stagecue ./cmd/stagecue

WORKDIR /data
ENTRYPOINT ["/out/stagecue"]
CMD ["--smoke-test"]
