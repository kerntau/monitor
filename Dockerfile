# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM node:22-alpine AS web
WORKDIR /src/internal/site
COPY internal/site/package.json internal/site/package-lock.json ./
RUN npm ci
COPY internal/site/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.27.1-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/internal/site/dist ./internal/site/dist
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -tags no_ui -trimpath -ldflags="-s -w" -o /monitor ./internal/cmd/hub

FROM scratch
COPY --from=build /monitor /monitor
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY LICENSE /LICENSE
ENV CHENV_MODE=true
VOLUME ["/monitor_data"]
EXPOSE 8090
ENTRYPOINT ["/monitor"]
CMD ["serve", "--http=0.0.0.0:8090", "--dir=/monitor_data"]
