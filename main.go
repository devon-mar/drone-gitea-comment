package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"text/template"
	"time"
)

var (
	template_funcs = template.FuncMap{"readEnv": os.Getenv, "readFile": readFile}
)

type CommentPayload struct {
	Body string `json:"body"`
}

type CommentResponse struct {
	HtmlUrl string `json:"html_url"`
}

func readFile(filename string) string {
	content, _ := os.ReadFile(filename)
	return string(content)
}

func getUrl(gitea_url string) string {
	return fmt.Sprintf(
		"%s/api/v1/repos/%s/%s/issues/%s/comments",
		gitea_url,
		// Assume these aren't empty since they're set by Drone.
		// PULL_REQUEST has already been checked in main
		os.Getenv("DRONE_REPO_OWNER"),
		os.Getenv("DRONE_REPO_NAME"),
		os.Getenv("DRONE_PULL_REQUEST"),
	)
}

func templatePayload() (*bytes.Buffer, error) {
	tmpl := template.New("body").Funcs(template_funcs)
	var err error

	plugin_body := os.Getenv("PLUGIN_BODY")
	plugin_body_file := os.Getenv("PLUGIN_BODY_FILE")
	if plugin_body != "" && plugin_body_file != "" {
		return nil, errors.New("body and body_file are mutually exclusive")
	} else if plugin_body != "" {
		tmpl, err = tmpl.Parse(plugin_body)
	} else if plugin_body_file != "" {
		var content []byte
		content, err = os.ReadFile(plugin_body_file)
		if err != nil {
			return nil, err
		}
		tmpl, err = tmpl.Parse(string(content))
	} else {
		return nil, errors.New("body OR body_file must be set")
	}

	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, nil)
	if err != nil {
		return nil, err
	}

	body_buf := &bytes.Buffer{}
	err = json.NewEncoder(body_buf).Encode(CommentPayload{Body: buf.String()})
	if err != nil {
		return nil, err
	}

	return body_buf, nil
}

func postComment() (string, error) {
	gitea_url := os.Getenv("PLUGIN_URL")
	if gitea_url == "" {
		return "", errors.New("url is not set")
	}

	gitea_token := os.Getenv("PLUGIN_TOKEN")
	if gitea_token == "" {
		return "", errors.New("token is not set")
	}

	if os.Getenv("DRONE_PULL_REQUEST") == "" {
		// What's the point if there's no PR
		return "", errors.New("empty DRONE_PULL_REQUEST")
	}

	payload, err := templatePayload()
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", getUrl(gitea_url), payload)
	req.Header.Add("Authorization", "token "+gitea_token)
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		return "", err
	}

	httpClient := http.Client{Timeout: time.Second * 10}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	} else if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("error posting comment (HTTP %d): %v", resp.StatusCode, err)
	}

	defer resp.Body.Close()
	cr := CommentResponse{}
	json.NewDecoder(resp.Body).Decode(&cr)

	return cr.HtmlUrl, nil
}

func main() {
	url, err := postComment()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Comment posted successfully: %s\n", url)
}
