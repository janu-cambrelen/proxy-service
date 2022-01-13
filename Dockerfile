# STAGE 1
FROM golang:alpine AS builder

RUN update-ca-certificates

ENV GROUP app-group
ENV USER app-user
ENV UID 10001

RUN mkdir /app
RUN addgroup "${GROUP}"
RUN adduser \
    --disabled-password \
    --gecos "" \
    --no-create-home \
    --home "$(pwd)" \
    --ingroup "${GROUP}" \
    --uid "${UID}" \
    "${USER}"

WORKDIR /app

COPY . .

RUN go mod download
RUN go mod verify

ENV CGO_ENABLED 0

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-w -s" -o /app/proxy

# STAGE 2
FROM scratch

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /app/proxy /
COPY --from=builder /app/.env /

USER app-user:app-group

EXPOSE 8080

ENTRYPOINT ["/proxy"]
