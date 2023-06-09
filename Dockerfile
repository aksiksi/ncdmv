FROM golang:latest as builder

WORKDIR /usr/src/ncdmv

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN make install

FROM chromedp/headless-shell:latest

# Override the default entrypoint.
RUN apt-get update; apt-get upgrade -y; apt install dumb-init -y
ENTRYPOINT ["dumb-init", "--"]

COPY --from=builder /usr/local/bin/ncdmv /usr/local/bin/ncdmv
# COPY --from=builder /usr/src/ncdmv/database/ncdmv.db /etc/ncdmv/ncdmv.db
ENV NCDMV_DISABLE_GPU=1
CMD ["ncdmv"]
