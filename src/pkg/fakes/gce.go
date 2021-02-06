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

// Package fakes contains fake implementations to be used in unit tests.
package fakes

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

// writeError writes the given error code to the given http.ResponseWriter.
func writeError(w http.ResponseWriter, r *http.Request, code int) {
	w.WriteHeader(code)
	resp := &googleapi.Error{Code: code}
	bytes, err := json.Marshal(resp)
	if err != nil {
		log.Printf("error: %s URL: %s Response: %v", err, r.URL.Path, resp)
		return
	}
	w.Write(bytes)
}

// GCE is a fake GCE implementation. It is intended to be constructed with NewGCEServer.
//
// The GCE struct represents the state of the fake GCE instance. Fields on this struct can be modified to influence the
// return values of GCE API calls.
//
// GCE should not be considered concurrency-safe. Do not use this struct in a concurrent way.
type GCE struct {
	// Images represents the images present in the project.
	Images *compute.ImageList
	// Deprecated represents the set of deprecated images in the project.
	Deprecated map[string]*compute.DeprecationStatus
	// Operations is the sequence of operations that the fake GCE server should return.
	Operations []*compute.Operation
	// server is an HTTP server that serves fake GCE requests. Requests are served using the state stored in
	// the other struct fields.
	server  *httptest.Server
	project string
}

// NewGCEServer constructs a fake GCE implementation for a given GCE project.
func NewGCEServer(project string) *GCE {
	gce := &GCE{
		Images:     &compute.ImageList{},
		Deprecated: make(map[string]*compute.DeprecationStatus),
		project:    project,
	}
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/projects/%s/global/images", project), gce.imagesListHandler)
	mux.HandleFunc(fmt.Sprintf("/projects/%s/global/images/", project), gce.imageHandler)
	mux.HandleFunc(fmt.Sprintf("/projects/%s/global/operations/", project), gce.operationsHandler)
	gce.server = httptest.NewServer(mux)
	return gce
}

// Client gets a GCE client to use for accessing the fake GCE server.
func (g *GCE) Client() (*compute.Service, error) {
	client, err := compute.New(g.server.Client())
	if err != nil {
		return nil, err
	}
	client.BasePath = g.server.URL
	return client, nil
}

func (g *GCE) operation() *compute.Operation {
	op := g.Operations[0]
	g.Operations = g.Operations[1:]
	return op
}

func (g *GCE) deprecate(name string, status *compute.DeprecationStatus) *compute.Operation {
	g.Deprecated[name] = status
	return g.operation()
}

func (g *GCE) image(name string) *compute.Image {
	for _, image := range g.Images.Items {
		if image.Name == name {
			return image
		}
	}
	return nil
}

func (g *GCE) imagesListHandler(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.Marshal(g.Images)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError)
		return
	}
	w.Write(bytes)
}

func (g *GCE) imageHandler(w http.ResponseWriter, r *http.Request) {
	// Path starts with /project/<project>/global/images/<name>
	splitPath := strings.Split(r.URL.Path, "/")
	splitPath = splitPath[1:]
	switch {
	case len(splitPath) == 5:
		image := g.image(splitPath[4])
		if image == nil {
			writeError(w, r, http.StatusNotFound)
			return
		}
		bytes, err := json.Marshal(image)
		if err != nil {
			writeError(w, r, http.StatusInternalServerError)
			return
		}
		w.Write(bytes)
	case len(splitPath) == 6 && splitPath[5] == "deprecate":
		if g.image(splitPath[4]) == nil {
			writeError(w, r, http.StatusNotFound)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("failed to read body")
			writeError(w, r, http.StatusInternalServerError)
			return
		}
		status := &compute.DeprecationStatus{}
		err = json.Unmarshal(body, status)
		if err != nil {
			log.Printf("failed to parse body: %s", string(body))
			writeError(w, r, http.StatusInternalServerError)
			return
		}
		op := g.deprecate(splitPath[4], status)
		bytes, err := json.Marshal(op)
		if err != nil {
			log.Printf("failed to marshal operation: %v", op)
			writeError(w, r, http.StatusInternalServerError)
			return
		}
		w.Write(bytes)
	default:
		log.Printf("unrecognized path: %s", r.URL.Path)
		writeError(w, r, http.StatusNotFound)
	}
}

func (g *GCE) operationsHandler(w http.ResponseWriter, r *http.Request) {
	bytes, err := json.Marshal(g.operation())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError)
		return
	}
	w.Write(bytes)
}

// Close closes the fake GCE server.
func (g *GCE) Close() {
	g.server.Close()
}

// GCEForTest encapsulates boilerplate needed for many test cases.
func GCEForTest(t *testing.T, project string) (*GCE, *compute.Service) {
	t.Helper()
	gce := NewGCEServer(project)
	client, err := gce.Client()
	if err != nil {
		t.Fatal(err)
	}
	return gce, client
}
