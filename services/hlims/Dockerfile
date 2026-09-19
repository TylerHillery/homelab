FROM --platform=$BUILDPLATFORM golang:1.26-alpine as build

WORKDIR /work

# Install git so that go build includes VCS details.
RUN apk add git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETOS TARGETARCH TARGETVARIANT
RUN \
  if [ "${TARGETARCH}" = "arm" ] && [ -n "${TARGETVARIANT}" ]; then \
  export GOARM="${TARGETVARIANT#v}"; \
  fi; \
  GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -v ./cmd/hlimsd

FROM gcr.io/distroless/static-debian12:nonroot

ENV HOME /home/nonroot

COPY --from=build /work/hlimsd /hlimsd
ENTRYPOINT ["/hlimsd"]
CMD ["--listen", "0.0.0.0:8080", "--sqlitedb", "/home/nonroot/hlims.db"]
