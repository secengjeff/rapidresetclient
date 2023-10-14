# Rapid Reset Client

Rapid Reset Client is a tool for testing mitigations and exposure to CVE-2023-44487 (Rapid Reset DDoS attack vector). It implements a minimal HTTP/2 client that opens a single TCP socket, negotiates TLS, ignores the certificate, and exchanges SETTINGS frames. The client then sends rapid HEADERS frames followed by RST_STREAM frames. It monitors for (but does not handle) server frames after initial setup, other than to send to stdout. This functionality is easily removed from source if it's too annoying. 

## Prerequisites

- [Go](https://golang.org/dl/)

Tested on go1.21.3 on arm64.  

## Installation

### Clone the Repository

```
git clone https://github.com/secengjeff/rapidresetclient.git
```

### Installing

```
cd rapidresetclient

go get golang.org/x/net/http2

go get golang.org/x/net/http2/hpack

go build -o rapidresetclient
```

### Flags

- `requests`: Number of requests to send (default is 5)

- `url`: Server URL (default is `https://localhost:443`)

- `wait`: Wait time in milliseconds between starting workers (default is 0)

- `concurrency`: Maximum number of concurrent workers (default is 0 for no limit)

### Example

Send 10 HTTP/2 requests (HEADERS and RST_STREAM frames) over a single connection to https://example.com using 5 workers and a wait time of 100 ms between each invocation.

```
./rapidresetclient  -requests=10  -url  https://example.com  -wait=100  -concurrency=5
```

## Built With

- [http2](https://pkg.go.dev/golang.org/x/net/http2) - Package that exposes low-level HTTP/2 primitives

- [http2/hpack](https://pkg.go.dev/golang.org/x/net/http2/hpack) - HTTP/2 header compression package

## Authors

-  Jeffrey  Lyon  -  *Initial  release*  - @secengjeff

See  also  the  list  of [contributors](https://github.com/secengjeff/rapidresetclient/contributors)

## License

This  project  is  licensed  under  the  Apache  License  -  see  the [LICENSE](LICENSE) file for details

## Acknowledgments

This  work  is  based  on  the [initial analysis  of  CVE-2023-44487](https://cloud.google.com/blog/products/identity-security/how-it-works-the-novel-http2-rapid-reset-ddos-attack) by  @jsnell  and  Daniele  Iamartino  at  Google.