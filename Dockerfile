FROM golang:1.18-alpine AS builder

# INSTALL NECESSARY DEPENDENCIES
RUN apk add --no-cache build-base git

# SET WORKING DIRECTORY
WORKDIR /app

# COPY GO MODULE FILES
COPY go.mod go.sum ./

# DOWNLOAD DEPENDENCIES
RUN go mod download

# COPY SOURCE CODE
COPY . .

# BUILD THE APPLICATION
RUN CGO_ENABLED=1 GOOS=linux go build -a -o crepes ./cmd/crepes

# CREATE RUNTIME IMAGE
FROM alpine:3.17

# INSTALL RUNTIME DEPENDENCIES
RUN apk add --no-cache ca-certificates chromium ffmpeg

# SET WORKING DIRECTORY
WORKDIR /app

# CREATE REQUIRED DIRECTORIES
RUN mkdir -p /app/data /app/storage /app/thumbnails

# COPY BINARY FROM BUILDER
COPY --from=builder /app/crepes /app/

# EXPOSE PORT
EXPOSE 8080

# SET ENVIRONMENT VARIABLES
ENV CHROMIUM_PATH=/usr/bin/chromium-browser

# RUN THE APPLICATION
CMD ["/app/crepes"]
