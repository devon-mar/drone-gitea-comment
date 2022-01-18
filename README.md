# drone-gitea-comment

A Drone plugin to template and leave a comment on a Gitea pull request.

## Settings
* `url`: The Gitea url. Example: `https://gitea.example.com` (REQURED).
* `token`: The Gitea token (REQUIRED).
* `body`: The body of the pull request or inline [Go template](https://golang.org/pkg/text/template/). One of `body` or `body_file` is required.
* `body_file`: Path to a file or [Go template](https://golang.org/pkg/text/template/). One of `body` or `body_file` is required.

## Template Functions
* `readFile`: Reads a file. Returns empty if there is an error reading the file.
* `readEnv`: Reads an environment variable. Returns empty if it is unset.

## Example:
```yaml
steps:
- name: Leave a comment (inline)
  image: devonm/drone-gitea-comment:latest
  settings:
    url: https://gitea.example.com
    token: mytoken
    body: |
      # The Title
      This is PR #{{ readEnv "DRONE_PULL_REQUEST" }}

      ## Build Output
      {{ readFile "output.txt" }}

- name: Leave a comment (using a template file)
  image: devonm/drone-gitea-comment:latest
  settings:
    url: https://gitea.example.com
    token:
      from_secret: gitea_token
    body_file: pr_comment.tmpl
```
