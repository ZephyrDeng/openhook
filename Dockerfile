FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/openhook ./cmd/openhook

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/openhook /app/openhook
ENV OPENHOOK_ADDR=:8080
ENV OPENHOOK_DB=/tmp/openhook.db
EXPOSE 8080
ENTRYPOINT ["/app/openhook"]
