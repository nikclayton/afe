FROM gcr.io/bitnami-containers/minideb-extras:jessie-r14

LABEL maintainer="Nik Clayton"

COPY lb.linux /app/lb
COPY config.yaml /app/config.yaml

USER root

WORKDIR /app

EXPOSE 8080

ENTRYPOINT ["/app/lb"]