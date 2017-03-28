# fu

fu is a personal temporary file upload client and server.

## Server

```sh
$ fu -s -addr=":3000" -token="secret"
```

## Client

```sh
$ export FU_ADDR="http://localhost:3000"
$ export FU_TOKEN="secret"
$ fu -d 2w main.go
$ exiftool -all= -o - ~/tmp/IMG_20160420_162000.jpg | fu -
```
