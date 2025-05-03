FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /gchat-devin

FROM gcr.io/distroless/static
WORKDIR /

COPY --from=builder /gchat-devin /gchat-devin

ENTRYPOINT ["/gchat-devin"]
