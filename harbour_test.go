package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"
	"time"
)

func TestGetAuthed(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("GET", mediaServer.URL+"/user/repo/objects/"+authedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Authorization", authedToken)
	req.Header.Set("Accept", contentMediaType)

	res, err := http.DefaultTransport.RoundTrip(req) // Do not follow the redirect
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 302 {
		t.Fatalf("expected status 302, got %d", res.StatusCode)
	}
}

func TestGetUnauthed(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("GET", mediaServer.URL+"/user/repo/objects/"+authedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Accept", contentMediaType)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 404 {
		t.Fatalf("expected status 404, got %d %s", res.StatusCode, req.URL)
	}
}

func TestGetMetaAuthed(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("GET", mediaServer.URL+"/user/repo/objects/"+authedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Authorization", authedToken)
	req.Header.Set("Accept", metaMediaType)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d %s", res.StatusCode, req.URL)
	}

	var meta Representation
	dec := json.NewDecoder(res.Body)
	dec.Decode(&meta)

	if meta.Oid != authedOid {
		t.Fatalf("expected to see oid `%s` in meta, got: `%s`", authedOid, meta.Oid)
	}

	if meta.Size != 42 {
		t.Fatalf("expected to see a size of `42`, got: `%d`", meta.Size)
	}

	download := meta.Links["download"]
	if download.Href != "http://localhost:8080/user/repo/objects/"+authedOid {
		t.Fatalf("expected download link, got %s", download.Href)
	}
}

func TestGetMetaUnauthed(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("GET", mediaServer.URL+"/user/repo/objects/"+authedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Accept", metaMediaType)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 404 {
		t.Fatalf("expected status 404, got %d", res.StatusCode)
	}

	var msg map[string]string
	dec := json.NewDecoder(res.Body)
	dec.Decode(&msg)

	if m := msg["message"]; m != "Not Found" {
		t.Fatalf("expected a message in the 404 json response")
	}
}

func TestPostAuthedNewObject(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("POST", mediaServer.URL+"/user/repo/objects", nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Authorization", authedToken)
	req.Header.Set("Accept", metaMediaType)

	buf := bytes.NewBufferString(fmt.Sprintf(`{"oid":"%s", "size":1234}`, nonexistingOid))
	req.Body = ioutil.NopCloser(buf)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 201 {
		t.Fatalf("expected status 201, got %d", res.StatusCode)
	}

	var meta Representation
	dec := json.NewDecoder(res.Body)
	dec.Decode(&meta)

	if meta.Oid != nonexistingOid {
		t.Fatalf("expected to see oid `%s` in meta, got: `%s`", nonexistingOid, meta.Oid)
	}

	if meta.Size != 1234 {
		t.Fatalf("expected to see a size of `1234`, got: `%d`", meta.Size)
	}

	download := meta.Links["download"]
	if download.Href != "http://localhost:8080/user/repo/objects/"+nonexistingOid {
		t.Fatalf("expected download link, got %s", download.Href)
	}

	upload, ok := meta.Links["upload"]
	if !ok {
		t.Fatal("expected upload link to be present")
	}

	if upload.Href != "https://examplebucket.s3.amazonaws.com"+oidPath(nonexistingOid) {
		t.Fatalf("expected upload link, got %s", upload.Href)
	}

	callback, ok := meta.Links["callback"]
	if !ok {
		t.Fatal("expected callback link to be present")
	}

	if callback.Href != "http://localhost:8080/user/repo/objects/"+nonexistingOid {
		t.Fatalf("expected callback link, got %s", callback.Href)
	}
}

func TestPostAuthedExistingObject(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("POST", mediaServer.URL+"/user/repo/objects", nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Authorization", authedToken)
	req.Header.Set("Accept", metaMediaType)

	buf := bytes.NewBufferString(fmt.Sprintf(`{"oid":"%s", "size":1234}`, authedOid))
	req.Body = ioutil.NopCloser(buf)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	var meta Representation
	dec := json.NewDecoder(res.Body)
	dec.Decode(&meta)

	if meta.Oid != authedOid {
		t.Fatalf("expected to see oid `%s` in meta, got: `%s`", authedOid, meta.Oid)
	}

	if meta.Size != 1234 {
		t.Fatalf("expected to see a size of `1234`, got: `%d`", meta.Size)
	}

	download := meta.Links["download"]
	if download.Href != "http://localhost:8080/user/repo/objects/"+authedOid {
		t.Fatalf("expected download link, got %s", download.Href)
	}

	upload, ok := meta.Links["upload"]
	if !ok {
		t.Fatalf("expected upload link to be present")
	}

	if upload.Href != "https://examplebucket.s3.amazonaws.com"+oidPath(authedOid) {
		t.Fatalf("expected upload link, got %s", upload.Href)
	}
}

func TestPostAuthedReadOnly(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("POST", mediaServer.URL+"/user/readonly/objects", nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Authorization", authedToken)
	req.Header.Set("Accept", metaMediaType)

	buf := bytes.NewBufferString(fmt.Sprintf(`{"oid":"%s", "size":1234}`, authedOid))
	req.Body = ioutil.NopCloser(buf)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 403 {
		t.Fatalf("expected status 403, got %d", res.StatusCode)
	}
}

func TestPostUnauthed(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("POST", mediaServer.URL+"/user/readonly/objects", nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Accept", metaMediaType)

	buf := bytes.NewBufferString(fmt.Sprintf(`{"oid":"%s", "size":1234}`, authedOid))
	req.Body = ioutil.NopCloser(buf)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 404 {
		t.Fatalf("expected status 404, got %d", res.StatusCode)
	}
}

func TestOptionsExistingObject(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("OPTIONS", mediaServer.URL+"/user/repo/objects/"+authedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Authorization", authedToken)
	req.Header.Set("Accept", contentMediaType)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("expected status code 200, got %d", res.StatusCode)
	}
}

func TestOptionsNonExistingObject(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("OPTIONS", mediaServer.URL+"/user/repo/objects/"+nonexistingOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Authorization", authedToken)
	req.Header.Set("Accept", contentMediaType)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 204 {
		t.Fatalf("expected status code 204, got %d", res.StatusCode)
	}
}

func TestOptionsUnauthed(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("OPTIONS", mediaServer.URL+"/user/repo/objects/"+authedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Accept", contentMediaType)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 404 {
		t.Fatalf("expected status code 404, got %d", res.StatusCode)
	}
}

func TestPut(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("PUT", mediaServer.URL+"/user/repo/objects/"+authedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Authorization", authedToken)
	req.Header.Set("Accept", contentMediaType)
	req.Header.Set("Content-Type", "application/octet-stream")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 405 {
		t.Fatalf("expected status 405, got %d", res.StatusCode)
	}
}

func TestCallbackWithSuccess(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("POST", mediaServer.URL+"/user/repo/objects/"+authedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Authorization", authedToken)
	req.Header.Set("Accept", metaMediaType)

	buf := bytes.NewBufferString(fmt.Sprintf(`{"oid":"%s", "status":200, "body":"ok"}`, authedOid))
	req.Body = ioutil.NopCloser(buf)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}
}

func TestMediaTypesRequired(t *testing.T) {
	testSetup()
	defer testTeardown()

	m := []string{"GET", "PUT", "OPTIONS"}
	for _, method := range m {
		req, err := http.NewRequest(method, mediaServer.URL+"/user/repo/objects/"+authedOid, nil)
		if err != nil {
			t.Fatalf("request error: %s", err)
		}
		req.Header.Set("Authorization", authedToken)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("response error: %s", err)
		}

		if res.StatusCode != 404 {
			t.Fatalf("expected status 404, got %d", res.StatusCode)
		}
	}
}

func TestMediaTypesParsed(t *testing.T) {
	testSetup()
	defer testTeardown()

	req, err := http.NewRequest("GET", mediaServer.URL+"/user/repo/objects/"+authedOid, nil)
	if err != nil {
		t.Fatalf("request error: %s", err)
	}
	req.Header.Set("Authorization", authedToken)
	req.Header.Set("Accept", contentMediaType+"; charset=utf-8")

	res, err := http.DefaultTransport.RoundTrip(req) // Do not follow the redirect
	if err != nil {
		t.Fatalf("response error: %s", err)
	}

	if res.StatusCode != 302 {
		t.Fatalf("expected status 302, got %d", res.StatusCode)
	}
}

var (
	now         time.Time
	mediaServer *httptest.Server
	metaServer  *httptest.Server

	authedOid      = "44ce7dd67c959e0d3524ffac1771dfbba87d2b6b4b4e99e42034a8b803f8b072"
	nonexistingOid = "aec070645fe53ee3b3763059376134f058cc337247c978add178b6ccdfb0019f"

	authedToken = "AUTHORIZED"
)

func testSetup() {
	Config.AwsKey = "AKIAIOSFODNN7EXAMPLE"
	Config.AwsSecretKey = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	Config.AwsBucket = "examplebucket"
	Config.Scheme = "http"

	contentSha = sha256Hex([]byte(content))
	now, _ = time.Parse(time.RFC822, "24 May 13 00:00 GMT")

	app := NewApp(&TestRedirector{}, &MetaStore{})
	mediaServer = httptest.NewServer(app.Router)
	metaServer = httptest.NewServer(newMetaServer())
	Config.MetaEndpoint = metaServer.URL

	logger = NewKVLogger(ioutil.Discard)
}

func testTeardown() {
	mediaServer.Close()
	metaServer.Close()
}

type TestRedirector struct {
}

func (t *TestRedirector) Get(meta *Meta, w http.ResponseWriter, r *http.Request) int {
	token := S3SignQuery("GET", path.Join("/", meta.PathPrefix, oidPath(meta.Oid)), 86400)
	w.Header().Set("Location", token.Location)
	w.WriteHeader(302)
	return 302
}

func (t *TestRedirector) PutLink(meta *Meta) *link {
	token := S3SignHeader("PUT", path.Join("/", meta.PathPrefix, oidPath(meta.Oid)), meta.Oid)
	header := make(map[string]string)
	header["Authorization"] = token.Token
	header["x-amz-content-sha256"] = meta.Oid
	header["x-amz-date"] = token.Time.Format(isoLayout)

	return &link{Href: token.Location, Header: header}
}

func (t *TestRedirector) Exists(*Meta) (bool, error) {
	return true, nil
}

func newMetaServer() http.Handler {
	router := NewRouter()
	s := router.Route("/{user}/{repo}/media/blobs/{oid}")

	s.Get(Config.ApiMediaType, func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if authz != authedToken {
			w.WriteHeader(404)
			return
		}

		vars := Vars(r)
		oid := vars["oid"]
		repo := vars["repo"]

		if oid == nonexistingOid {
			if repo == "readonly" {
				fmt.Fprint(w, `{"writeable":false}`)
			} else {
				fmt.Fprint(w, `{"writeable":true}`)
			}
		} else {
			if repo == "readonly" {
				fmt.Fprintf(w, `{"oid":"%s","size":42,"writeable":false}`, oid)
			} else {
				fmt.Fprintf(w, `{"oid":"%s","size":42,"writeable":true}`, oid)
			}
		}
	})

	s.Post(Config.ApiMediaType, func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if authz != authedToken {
			w.WriteHeader(404)
			return
		}

		vars := Vars(r)
		repo := vars["repo"]

		var m Representation
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&m)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		if repo == "readonly" {
			w.WriteHeader(403)
			return
		}

		if m.Oid == nonexistingOid {
			w.WriteHeader(201)
		}

		fmt.Fprintf(w, `{"oid":"%s","size":%d,"writeable":true}`, m.Oid, m.Size)
	})

	return router
}
