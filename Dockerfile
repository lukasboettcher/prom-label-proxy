FROM quay.io/prometheus/golang-builder:1.22-base as builder

COPY . .

RUN make

FROM quay.io/prometheus/busybox-linux-amd64:glibc

COPY --from=builder  /app/prom-label-proxy /bin/prom-label-proxy

USER        nobody
ENTRYPOINT  [ "/bin/prom-label-proxy" ]
