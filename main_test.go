package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func unsetAll() {
	all_vars := []string{
		"PLUGIN_URL", "PLUGIN_TOKEN", "DRONE_PULL_REQUEST",
		"PLUGIN_BODY", "PLUGIN_BODY_FILE", "DRONE_REPO_OWNER",
		"DRONE_REPO_NAME", "DRONE_PULL_REQUEST",
	}
	for _, v := range all_vars {
		os.Unsetenv(v)
	}
}

func setEnv(vars map[string]string) {
	for k, v := range vars {
		os.Setenv(k, v)
	}
}

func getTestServer(t *testing.T, token string, owner string, repo string, pr string, respUrl string, wantBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "token "+token {
			t.Errorf("Unexpected Authorization header: %q", auth)
			w.WriteHeader(http.StatusForbidden)
		} else if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Unexpected content-type: %q", ct)
			w.WriteHeader(http.StatusUnprocessableEntity)
		} else if r.URL.Path == fmt.Sprintf("/api/v1/repos/%s/%s/issues/%s/comments", owner, repo, pr) {
			cp := CommentPayload{}
			_ = json.NewDecoder(r.Body).Decode(&cp)
			if cp.Body != wantBody {
				t.Errorf("Wanted body %q but got %q", wantBody, cp.Body)
			}
			w.WriteHeader(http.StatusCreated)

			_ = json.NewEncoder(w).Encode(&CommentResponse{HtmlUrl: respUrl})
			w.Header().Add("Content-Type", "application/json")
		} else {
			t.Errorf("Unexpected path: %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestEnvErrors(t *testing.T) {
	cases := []map[string]string{
		{},
		// No URL
		{"DRONE_PULL_REQUEST": "1", "PLUGIN_BODY": "a", "PLUGIN_BODY_FILE": "file"},
		// No token
		{"PLUGIN_URL": "a", "DRONE_PULL_REQUEST": "1", "PLUGIN_BODY": "a", "PLUGIN_BODY_FILE": "file"},
		// No body variable is set
		{"PLUGIN_URL": "a", "PLUGIN_TOKEN": "b", "DRONE_PULL_REQUEST": "1"},
		// Both body variables are set
		{
			"PLUGIN_URL": "a", "PLUGIN_TOKEN": "b", "DRONE_PULL_REQUEST": "1",
			"PLUGIN_BODY": "a", "PLUGIN_BODY_FILE": "file",
		},
	}
	for i, tc := range cases {
		unsetAll()
		setEnv(tc)
		_, err := postComment()
		if err == nil {
			t.Errorf("[%d] Expected an error", i)
		}
	}
}

func TestInlineTemplate(t *testing.T) {
	unsetAll()

	owner := "testOnwer"
	repo := "testRepo"
	pr := "123"
	token := "s3crett0ken"
	wantBody := "this is a test"
	wantUrl := "http://success"

	server := getTestServer(t, token, owner, repo, pr, wantUrl, wantBody)
	defer server.Close()

	setEnv(map[string]string{
		"PLUGIN_URL":         server.URL,
		"PLUGIN_TOKEN":       token,
		"DRONE_REPO_OWNER":   owner,
		"DRONE_REPO_NAME":    repo,
		"DRONE_PULL_REQUEST": pr,
		"MY_TEST_VAR":        wantBody,
		"PLUGIN_BODY":        `{{ readEnv "MY_TEST_VAR" }}`,
	})

	url, err := postComment()
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	} else if url != wantUrl {
		t.Errorf("Expected URL %q but got %q", wantUrl, url)
	}
}

func TestFileTemplate(t *testing.T) {
	unsetAll()

	owner := "testOnwer"
	repo := "testRepo"
	pr := "123"
	token := "s3crett0ken"
	wantBody := "read from the environment\nread from a file"
	wantUrl := "http://success2"

	server := getTestServer(t, token, owner, repo, pr, wantUrl, wantBody)
	defer server.Close()

	setEnv(map[string]string{
		"PLUGIN_URL":         server.URL,
		"PLUGIN_TOKEN":       token,
		"DRONE_REPO_OWNER":   owner,
		"DRONE_REPO_NAME":    repo,
		"DRONE_PULL_REQUEST": pr,
		"MY_TEST_VAR2":       "read from the environment",
		"PLUGIN_BODY_FILE":   "./testdata/test.tmpl",
	})

	url, err := postComment()
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	} else if url != wantUrl {
		t.Errorf("Expected URL %q but got %q", wantUrl, url)
	}
}

func TestInvalidTemplateFile(t *testing.T) {
	unsetAll()

	setEnv(map[string]string{
		"PLUGIN_URL":         "url",
		"PLUGIN_TOKEN":       "token",
		"DRONE_REPO_OWNER":   "owner",
		"DRONE_REPO_NAME":    "repo",
		"DRONE_PULL_REQUEST": "123",
		"PLUGIN_BODY_FILE":   "./testdata/invalid.tmpl",
	})

	_, err := postComment()
	if err == nil {
		t.Error("Expected an error")
	}
}
