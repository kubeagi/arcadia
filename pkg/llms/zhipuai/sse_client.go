/*
Copyright 2023 KubeAGI.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// NOTE: Reference https://github.com/r3labs/sse/client.go
// NOTE: Reference zhipuai's python sdk: utils/sse_client.py

package zhipuai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/r3labs/sse/v2"
)

func defaultHandler(event *sse.Event) {
	switch string(event.Event) {
	case "add", "error", "interrupted":
		fmt.Printf("%s", event.Data)
	default:
		fmt.Printf("%s", event.Data)
	}
}
func Stream(apiURL, token string, params ModelParams, timeout time.Duration, handler func(*sse.Event)) error {
	// normal post request
	jsonParams, err := json.Marshal(params)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonParams))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", token)

	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Printf("%v", resp)
	// parse response body as stream events
	eventChan, errorChan := NewSSEClient().Events(resp)

	// handle events
	if handler == nil {
		handler = defaultHandler
	}

	for {
		select {
		case err = <-errorChan:
			return err
		case msg := <-eventChan:
			handler(msg)
		}
	}
}

var (
	headerID    = []byte("id:")
	headerData  = []byte("data:")
	headerEvent = []byte("event:")
	headerRetry = []byte("retry:")
)

type SSEClient struct {
	LastEventID    atomic.Value // []byte
	EncodingBase64 bool

	maxBufferSize int
}

func NewSSEClient() *SSEClient {
	return &SSEClient{
		maxBufferSize: 1 << 16,
	}
}

func (c *SSEClient) Events(resp *http.Response) (<-chan *sse.Event, <-chan error) {
	reader := sse.NewEventStreamReader(resp.Body, c.maxBufferSize)
	return c.startReadLoop(reader)
}

func (c *SSEClient) startReadLoop(reader *sse.EventStreamReader) (chan *sse.Event, chan error) {
	outCh := make(chan *sse.Event)
	erChan := make(chan error)
	go c.readLoop(reader, outCh, erChan)
	return outCh, erChan
}

func (c *SSEClient) readLoop(reader *sse.EventStreamReader, outCh chan *sse.Event, erChan chan error) {
	for {
		// Read each new line and process the type of event
		event, err := reader.ReadEvent()
		if err != nil {
			if err == io.EOF {
				erChan <- nil
				return
			}
			erChan <- err
			return
		}

		// If we get an error, ignore it.
		var msg *sse.Event
		if msg, err = c.processEvent(event); err == nil {
			if len(msg.ID) > 0 {
				c.LastEventID.Store(msg.ID)
			} else {
				msg.ID, _ = c.LastEventID.Load().([]byte)
			}

			// Send downstream if the event has something useful
			if hasContent(msg) {
				outCh <- msg
			}
		}
	}
}

func (c *SSEClient) processEvent(msg []byte) (event *sse.Event, err error) {
	var e sse.Event

	if len(msg) < 1 {
		return nil, errors.New("event message was empty")
	}

	// Normalize the crlf to lf to make it easier to split the lines.
	// Split the line by "\n" or "\r", per the spec.
	for _, line := range bytes.FieldsFunc(msg, func(r rune) bool { return r == '\n' || r == '\r' }) {
		switch {
		case bytes.HasPrefix(line, headerID):
			e.ID = append([]byte(nil), trimHeader(len(headerID), line)...)
		case bytes.HasPrefix(line, headerData):
			// The spec allows for multiple data fields per event, concatenated them with "\n".
			e.Data = append(e.Data, append(trimHeader(len(headerData), line), byte('\n'))...)
		// The spec says that a line that simply contains the string "data" should be treated as a data field with an empty body.
		case bytes.Equal(line, bytes.TrimSuffix(headerData, []byte(":"))):
			e.Data = append(e.Data, byte('\n'))
		case bytes.HasPrefix(line, headerEvent):
			e.Event = append([]byte(nil), trimHeader(len(headerEvent), line)...)
		case bytes.HasPrefix(line, headerRetry):
			e.Retry = append([]byte(nil), trimHeader(len(headerRetry), line)...)
		default:
			// Ignore any garbage that doesn't match what we're looking for.
		}
	}

	// Trim the last "\n" per the spec.
	e.Data = bytes.TrimSuffix(e.Data, []byte("\n"))

	if c.EncodingBase64 {
		buf := make([]byte, base64.StdEncoding.DecodedLen(len(e.Data)))

		n, err := base64.StdEncoding.Decode(buf, e.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode event message: %s", err)
		}
		e.Data = buf[:n]
	}
	return &e, err
}

func hasContent(e *sse.Event) bool {
	return len(e.ID) > 0 || len(e.Data) > 0 || len(e.Event) > 0 || len(e.Retry) > 0
}

func trimHeader(size int, data []byte) []byte {
	if data == nil || len(data) < size {
		return data
	}

	data = data[size:]
	// Remove optional leading whitespace
	if len(data) > 0 && data[0] == 32 {
		data = data[1:]
	}
	// Remove trailing new line
	if len(data) > 0 && data[len(data)-1] == 10 {
		data = data[:len(data)-1]
	}
	return data
}
