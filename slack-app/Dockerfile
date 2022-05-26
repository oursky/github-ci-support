FROM golang:1.18-bullseye AS build

WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY *.go ./
RUN go build -o /github-slack-app

FROM gcr.io/distroless/base-debian11

WORKDIR /
COPY --from=build /github-slack-app /github-slack-app
EXPOSE 80
USER nonroot:nonroot

ENTRYPOINT ["/github-slack-app"]
