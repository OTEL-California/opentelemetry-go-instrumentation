FROM golang:1.24.0@sha256:cd0c949a4709ef70a8dad14274f09bd07b25542de5a1c4812f217087737efd17
WORKDIR /app
COPY . .
ARG BINARY_NAME
ENV BINARY_NAME=$BINARY_NAME
RUN go build -o $BINARY_NAME
ENTRYPOINT ["sh", "-c", "/app/$BINARY_NAME"]
