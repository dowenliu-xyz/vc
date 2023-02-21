FROM golang:latest AS builder
ARG GOPROXY
WORKDIR /opt/vc
COPY go.* ./
RUN GOPROXY=${GOPROXY} go mod download
COPY sub ./sub
COPY vc ./vc
COPY main.go ./
RUN GOPROXY=${GOPROXY} go build -o app .

FROM ubuntu:latest
WORKDIR /opt/vc
ENV TZ="Asia/Shanghai"
ENV V2RAY_CONFIG=/opt/v2ray/config.json
ENV VC_SUB_URL=""
ENV VC_SUB_CHECK=on
ENV VC_CHECK_PERIOD=60
ENV V2RAY_ASSET=/opt/v2ray/asset
ENV V2RAY_BIN=/opt/v2ray/v2ray
ENV VC_CHECK_TIMEOUT=5
ENV VC_CHECK_URL="https://httpbin.org/get"
ENV VC_API_PORT=3001
COPY --from=builder /opt/vc/app /opt/vc/vc
COPY core-pkg.sh /opt/vc/core-pkg.sh
RUN apt update && \
    apt install -y ca-certificates unzip curl && \
    update-ca-certificates && \
    chmod +x /opt/vc/core-pkg.sh &&  \
    /opt/vc/core-pkg.sh && \
    apt remove -y ca-certificates unzip curl && \
    apt clean -y && \
    rm -rf /var/lib/apt/lists
CMD ["./vc"]
