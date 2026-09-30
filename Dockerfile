# Build stage.
FROM golang:1.26-alpine AS build
WORKDIR /src

# Cache module downloads before copying the source.
COPY go.mod go.sum ./
RUN go mod download

# Build a static binary.
COPY . .
RUN CGO_ENABLED=0 go build -o /out/manager-backend .

# Runtime stage.
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/manager-backend ./manager-backend
EXPOSE 8000
ENTRYPOINT ["./manager-backend"]
