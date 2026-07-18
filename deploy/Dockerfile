FROM node:24-alpine AS web-build
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
ARG VITE_SOURCE_CODE_URL
ENV VITE_SOURCE_CODE_URL=$VITE_SOURCE_CODE_URL
RUN npm run build

FROM golang:1.26.5-alpine AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-build /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/dz-ai-creator ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/dz-ai-creator-admin ./cmd/admin \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/dz-ai-creator-secrets ./cmd/secrets

FROM alpine:3.22
RUN addgroup -S dzcreator && adduser -S -G dzcreator dzcreator
WORKDIR /app
COPY --from=go-build /out/dz-ai-creator /usr/local/bin/dz-ai-creator
COPY --from=go-build /out/dz-ai-creator-admin /usr/local/bin/dz-ai-creator-admin
COPY --from=go-build /out/dz-ai-creator-secrets /usr/local/bin/dz-ai-creator-secrets
COPY --from=web-build /src/web/dist ./web/dist
RUN mkdir -p /app/data/assets && chown -R dzcreator:dzcreator /app
USER dzcreator
EXPOSE 8888
ENTRYPOINT ["dz-ai-creator"]
