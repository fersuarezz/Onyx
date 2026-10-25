FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /onyx .

FROM alpine:3.20
RUN adduser -D -u 10001 onyx \
    && mkdir -p /data/media \
    && chown onyx /data/media
COPY --from=build /onyx /usr/local/bin/onyx
USER onyx
EXPOSE 8080
ENTRYPOINT ["onyx"]
