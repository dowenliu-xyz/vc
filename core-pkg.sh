#!/usr/bin/env bash
if [ -z "${RELEASE}" ]; then
  RELEASE=v4.45.2
fi
ARCH=$(uname -i)
case ${ARCH} in
  "aarch64" | "arm64")
    ARCH="arm64-v8a"
    ;;
  "x86_64")
    ARCH="64"
    ;;
  *)
    echo "only amd64/x86_64 and arm64/aarch64 platform are supported."
    exit 1
    ;;
esac
SAVE_DIR=$(mktemp -d)
echo "SAVE_DIR: ${SAVE_DIR}"
curl -L -o ${SAVE_DIR}/v2ray.zip https://github.com/v2fly/v2ray-core/releases/download/${RELEASE}/v2ray-linux-${ARCH}.zip
unzip ${SAVE_DIR}/v2ray.zip -d ${SAVE_DIR}/v2ray
mkdir -p /opt/v2ray/asset
mv ${SAVE_DIR}/v2ray/v2ray /opt/v2ray/v2ray
chmod +x /opt/v2ray/asset
rm -rf ${SAVE_DIR}
curl -L -o /opt/v2ray/asset/geoip.dat https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geoip.dat
curl -L -o /opt/v2ray/asset/geosite.dat https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/geosite.dat
