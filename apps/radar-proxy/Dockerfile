FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/radar-proxy ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates

WORKDIR /app
COPY --from=builder /app/radar-proxy /app/radar-proxy
COPY ../../scripts/projection_matrix.json /app/scripts/projection_matrix.json
COPY ../../scripts/seed_prompts.json /app/scripts/seed_prompts.json

EXPOSE 8081
ENV PORT=8081

CMD ["/app/radar-proxy"]
