FROM golang:1.17 AS build

WORKDIR /src
# enable modules caching in separate layer
COPY go.mod go.sum ./
RUN go mod download
COPY . ./

# HACK: force reachability status to be set to true.
ENV REACHABILITY_OVERRIDE_PUBLIC true

RUN make binary

FROM debian:11.2-slim

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates procps dumb-init; \
    apt-get clean; \
    rm -rf /var/lib/apt/lists/*; \
    groupadd -r bee --gid 999; \
    useradd -r -g bee --uid 999 --no-log-init -m bee;

# make sure mounted volumes have correct permissions
RUN mkdir -p /home/bee/.bee && chown 999:999 /home/bee/.bee

COPY --from=build /src/dist/bee /usr/local/bin/bee
COPY --from=build /src/scripts/ContinueStopSwarm.sh /ContinueStopSwarm.sh
COPY --from=build /src/scripts/start.sh /start.sh

EXPOSE 1633 1634 1635
USER bee
WORKDIR /home/bee
VOLUME /home/bee/.bee

ENTRYPOINT ["/usr/bin/dumb_init", "--"]
CMD ["bee"]
