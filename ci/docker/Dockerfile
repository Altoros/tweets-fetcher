FROM golang:1.7

RUN set -x && \
  apt-get update && \
  apt-get install -y --no-install-recommends \
  curl jq wget git build-essential

RUN \
  curl -v -L -o /usr/local/bin/spiff https://github.com/cloudfoundry-incubator/spiff/releases/download/v1.0.7/spiff_linux_amd64 && \
  chmod +x /usr/local/bin/spiff

RUN \
  curl -v -L -o /usr/local/bin/spruce https://github.com/geofffranks/spruce/releases/download/v1.8.0/spruce-linux-amd64 && \
  chmod +x /usr/local/bin/spruce

RUN \
  curl -v -L -o /usr/local/bin/dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.1.3/dumb-init_1.1.3_amd64 && \
  chmod +x /usr/local/bin/dumb-init

RUN \
  curl -v -L -o ./cf.tgz https://s3.amazonaws.com/go-cli/releases/v6.21.1/cf-cli_6.21.1_linux_x86-64.tgz && \
  tar xzvf ./cf.tgz -C /usr/local/bin/ && \
  chmod +x /usr/local/bin/cf && \
  rm -f ./cf.tgz
