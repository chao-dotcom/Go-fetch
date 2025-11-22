package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const defaultAPIBase = "http://localhost:8080/v1"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "submit":
		err = submitCmd(args)
	case "job":
		err = jobCmd(args)
	case "list":
		err = listCmd(args)
	case "workers":
		err = simpleGet("/workers")
	case "queues":
		err = simpleGet("/queues")
	case "logs":
		err = logsCmd(args)
	case "cancel":
		err = actionCmd(args, "/jobs/%s")
	case "retry":
		err = actionCmd(args, "/jobs/%s/retry")
	case "health":
		err = simpleGet("/health")
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", cmd)
		printUsage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func submitCmd(args []string) error {
	fs := flag.NewFlagSet("submit", flag.ExitOnError)
	jobType := fs.String("type", "", "job type (required)")
	payloadRaw := fs.String("payload", "", "JSON payload string (required)")
	priority := fs.Int("priority", 0, "job priority")
	maxAttempts := fs.Int("max_attempts", 3, "maximum attempts")
	fs.Parse(args)
	if *jobType == "" || *payloadRaw == "" {
		return fmt.Errorf("type and payload are required")
	}

	var payload any
	if err := json.Unmarshal([]byte(*payloadRaw), &payload); err != nil {
		return fmt.Errorf("invalid payload JSON: %w", err)
	}

	body := map[string]any{
		"type":         *jobType,
		"payload":      payload,
		"priority":     *priority,
		"max_attempts": *maxAttempts,
	}
	return doRequest("POST", "/jobs", body)
}

func jobCmd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: taskqueue-cli job <job_id>")
	}
	route := fmt.Sprintf("/jobs/%s", url.PathEscape(args[0]))
	return simpleGet(route)
}

func listCmd(args []string) error {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	status := fs.String("status", "", "status filter")
	jobType := fs.String("type", "", "job type filter")
	limit := fs.Int("limit", 50, "limit")
	offset := fs.Int("offset", 0, "offset")
	fs.Parse(args)

	params := url.Values{}
	if *status != "" {
		params.Set("status", *status)
	}
	if *jobType != "" {
		params.Set("type", *jobType)
	}
	params.Set("limit", fmt.Sprintf("%d", *limit))
	params.Set("offset", fmt.Sprintf("%d", *offset))
	route := "/jobs"
	if enc := params.Encode(); enc != "" {
		route += "?" + enc
	}
	return simpleGet(route)
}

func logsCmd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: taskqueue-cli logs <job_id>")
	}
	route := fmt.Sprintf("/jobs/%s/logs", url.PathEscape(args[0]))
	return simpleGet(route)
}

func actionCmd(args []string, pattern string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: taskqueue-cli <cancel|retry> <job_id>")
	}
	route := fmt.Sprintf(pattern, url.PathEscape(args[0]))
	return doRequest("POST", route, nil)
}

func simpleGet(route string) error {
	return doRequest("GET", route, nil)
}

func doRequest(method, route string, body any) error {
	base := os.Getenv("API_BASE_URL")
	if base == "" {
		base = defaultAPIBase
	}
	target := strings.TrimRight(base, "/") + route

	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequest(method, target, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token := os.Getenv("API_BEARER_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s %s returned %d: %s", method, route, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var pretty bytes.Buffer
	if json.Indent(&pretty, respBody, "", "  ") == nil {
		fmt.Println(pretty.String())
	} else {
		fmt.Println(string(respBody))
	}
	return nil
}

func printUsage() {
	fmt.Println(`taskqueue-cli - simple client for the Task Queue API

Usage:
  taskqueue-cli submit -type <type> -payload '<json>' [-priority N] [-max_attempts N]
  taskqueue-cli job <job_id>
  taskqueue-cli list [-status STATUS] [-type TYPE] [-limit N] [-offset N]
  taskqueue-cli queues
  taskqueue-cli workers
  taskqueue-cli logs <job_id>
  taskqueue-cli cancel <job_id>
  taskqueue-cli retry <job_id>
  taskqueue-cli health

Environment variables:
  API_BASE_URL      Base URL for the API (default http://localhost:8080/v1)
  API_BEARER_TOKEN  Optional bearer token for authenticated requests`)
}

