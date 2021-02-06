// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fakes

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type gcsObject struct{ Name, Bucket string }
type gcsObjects struct{ Items []gcsObject }

func setTransportAddr(transport *http.Transport, addr string) {
	transport.DialTLS = func(_, _ string) (net.Conn, error) {
		return tls.Dial("tcp", addr, transport.TLSClientConfig)
	}
}

// GCS contains data and functionality for a fake GCS server. It is intended to be constructed with NewGCSServer.
//
// The GCS struct represents the state of the fake GCS instance. Fields on this struct can be modified to influence the
// return values of GCS API calls.
//
// The fake GCS server implements a small part of the API discussed here:
// https://godoc.org/cloud.google.com/go/storage. Only the parts that we need for testing
// are implemented here. Documentation for the GCS JSON API is here:
// https://cloud.google.com/storage/docs/json_api/v1/
//
// This struct should not be considered concurrency safe.
type GCS struct {
	// Objects represents the collection of objects that exist in the fake GCS server.
	// Keys are strings of the form "/<bucket>/<object path>". Values are data that belong
	// in each object.
	Objects map[string][]byte
	// Client is the client to use when accessing the fake GCS server.
	Client *storage.Client
	// Server is the fake GCS server. It uses state from this struct for serving requests.
	Server *httptest.Server
}

// NewGCSServer constructs a fake GCS implementation.
func NewGCSServer(ctx context.Context) (*GCS, error) {
	var err error
	gcs := &GCS{make(map[string][]byte), nil, nil}
	mux := http.NewServeMux()
	mux.HandleFunc("/", gcs.objectHandler)
	mux.HandleFunc("/storage/v1/b/", gcs.bucketHandler)
	mux.HandleFunc("/upload/storage/v1/b/", gcs.uploadHandler)
	gcs.Server = httptest.NewTLSServer(mux)
	httpClient := gcs.Server.Client()
	setTransportAddr(httpClient.Transport.(*http.Transport), gcs.Server.Listener.Addr().String())
	gcs.Client, err = storage.NewClient(ctx, option.WithHTTPClient(httpClient), option.WithoutAuthentication())
	if err != nil {
		gcs.Server.Close()
		return nil, err
	}
	return gcs, nil
}

func (g *GCS) objectHandler(w http.ResponseWriter, r *http.Request) {
	data, ok := g.Objects[r.URL.Path]
	if !ok {
		writeError(w, r, http.StatusNotFound)
		return
	}
	if _, err := w.Write(data); err != nil {
		log.Printf("write %q failed: %v", r.URL.Path, err)
	}
}

// list handles a `list` request.
// See: https://cloud.google.com/storage/docs/json_api/v1/#Objects, `list` method.
// Only handles the 'prefix' optional parameter.
func (g *GCS) list(w http.ResponseWriter, r *http.Request, bucket string) {
	if err := r.ParseForm(); err != nil {
		log.Printf("failed to parse form %q: %v", r.URL.Path, err)
		return
	}
	bucketPrefix := fmt.Sprintf("/%s/", bucket)
	prefix := bucketPrefix + r.Form.Get("prefix")
	var all gcsObjects
	for k := range g.Objects {
		if strings.HasPrefix(k, prefix) {
			all.Items = append(all.Items, gcsObject{strings.TrimPrefix(k, bucketPrefix), bucket})
		}
	}
	bytes, err := json.Marshal(all)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(bytes); err != nil {
		log.Printf("write %q failed: %v", r.URL.Path, err)
	}
}

// del handles a `delete` request.
// See: https://cloud.google.com/storage/docs/json_api/v1/#Objects, `delete` method.
// Doesn't handle any optional parameters.
func (g *GCS) del(w http.ResponseWriter, r *http.Request, bucket, objectPath string) {
	key := fmt.Sprintf("/%s/%s", bucket, objectPath)
	if _, ok := g.Objects[key]; !ok {
		log.Printf("delete failed: item %s does not exist", key)
		writeError(w, r, http.StatusNotFound)
		return
	}
	delete(g.Objects, key)
}

func (g *GCS) bucketHandler(w http.ResponseWriter, r *http.Request) {
	// Path looks like:
	// - /storage/v1/b/<bucket>/o
	// - /storage/v1/b/<bucket>/o/<object>
	splitPath := strings.SplitN(r.URL.Path, "/", 7)
	if len(splitPath) < 6 || splitPath[5] != "o" {
		log.Printf("unrecognized path: %s", r.URL.Path)
		writeError(w, r, http.StatusNotFound)
		return
	}
	objectPath := ""
	if len(splitPath) == 7 {
		objectPath = splitPath[6]
	}
	bucket := splitPath[4]
	switch {
	case objectPath != "" && r.Method == "DELETE":
		g.del(w, r, bucket, objectPath)
	case objectPath == "":
		g.list(w, r, bucket)
	default:
		log.Printf("unrecognized path: %s", r.URL.Path)
		writeError(w, r, http.StatusNotFound)
		return
	}
}

func (g *GCS) uploadHandler(w http.ResponseWriter, r *http.Request) {
	splitPath := strings.Split(r.URL.Path, "/")
	// Path looks like /upload/storage/v1/b/<bucket>/o
	// (see: https://cloud.google.com/storage/docs/json_api/v1/#Objects, `insert` method)
	// This implementation only accepts the path parameter 'bucket'. It does not accept any optional parameters.
	//
	// GCS uses multipart HTTP messages to upload data. The first part contains object metadata (name, bucket, etc)
	// in JSON format, and the second part contains the object data. Here, we extract the object metadata and data
	// from the multipart message and store it
	if len(splitPath) != 7 || splitPath[6] != "o" {
		log.Printf("unrecognized path: %s", r.URL.Path)
		writeError(w, r, http.StatusNotFound)
		return
	}
	_, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		log.Printf("failed to parse Content-Type: %s", r.Header.Get("Content-Type"))
		writeError(w, r, http.StatusInternalServerError)
		return
	}
	var parts [][]byte
	mr := multipart.NewReader(r.Body, params["boundary"])
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("failed to parse request: %v", r)
			writeError(w, r, http.StatusInternalServerError)
			return
		}
		partData, err := ioutil.ReadAll(part)
		if err != nil {
			log.Printf("failed to parse request: %v", r)
			writeError(w, r, http.StatusInternalServerError)
			return
		}
		parts = append(parts, partData)
	}
	objectMetadata := parts[0]
	objectData := parts[1]
	object := &gcsObject{}
	if err := json.Unmarshal(objectMetadata, object); err != nil {
		log.Printf("failed to parse object: %s", string(objectMetadata))
		writeError(w, r, http.StatusInternalServerError)
		return
	}
	g.Objects[fmt.Sprintf("/%s/%s", object.Bucket, object.Name)] = objectData
	if _, err := w.Write(objectMetadata); err != nil {
		log.Printf("write %q failed: %v", r.URL.Path, err)
	}
}

// Close closes the fake GCS server and its client.
func (g *GCS) Close() error {
	defer g.Server.Close()
	return g.Client.Close()
}

// GCSForTest encapsulates boilerplate for getting a GCS object in tests.
func GCSForTest(t *testing.T) *GCS {
	t.Helper()
	gcs, err := NewGCSServer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return gcs
}
