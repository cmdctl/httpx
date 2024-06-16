package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func main() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	// parse the input as an http request
	req, err := ParseRequest(WithEnvVars(input))
	if err != nil {
		log.Fatal(fmt.Errorf("failed to parse request: %w", err))
	}
	// send the request to the server
	resp, err := SendRequest(req)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to send request: %w", err))
	}

	// print the response
	_, err = os.Stdout.Write([]byte(resp))
	if err != nil {
		log.Fatal(err)
	}
}

// WithEnvVars replaces all environment variables in the input with their values
func WithEnvVars(input []byte) []byte {
	envVars := os.Environ()
	for _, envVar := range envVars {
		parts := strings.Split(envVar, "=")
		input = bytes.ReplaceAll(input, []byte(fmt.Sprintf("{{%s}}", parts[0])), []byte(parts[1]))
	}
	return input
}

func ParseRequest(req []byte) (http.Request, error) {
	var request http.Request

	scanner := bufio.NewScanner(bytes.NewReader(req))
	// scan the first line
	for scanner.Scan() {
		text := scanner.Text()
		text = strings.TrimSpace(text)
		if text != "" && text != "\n" && !strings.HasPrefix(text, "#") {
			break
		}
	}
	firstLine := scanner.Text()
	// parse the first line
	method, urlstr, err := ParseFirstLine(firstLine)
	if err != nil {
		return http.Request{}, err
	}
	// set the method and url
	request.Method = method
	request.URL, err = url.Parse(urlstr)
	if err != nil {
		return http.Request{}, err
	}

	// parse the headers
	headers := make(http.Header)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if line == "" {
			break
		}
		parts := strings.Split(line, ": ")
		if len(parts) != 2 {
			return http.Request{}, fmt.Errorf("invalid header: %s", line)
		}
		headers.Add(parts[0], parts[1])
	}

	// set the headers
	request.Header = headers

	// parse the body
	body := ""
	for scanner.Scan() {
		body += scanner.Text()
	}
	request.Body = io.NopCloser(strings.NewReader(body))
	return request, nil
}

func ParseFirstLine(firstLine string) (string, string, error) {
	// split the first line by spaces
	parts := strings.Split(firstLine, " ")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid first line: %s", firstLine)
	}
	return parts[0], parts[1], nil
}

func SendRequest(req http.Request) (string, error) {
	client := &http.Client{}
	resp, err := client.Do(&req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	response := fmt.Sprintf("HTTP/1.1 %d %s\n\n", resp.StatusCode, http.StatusText(resp.StatusCode))
	for key, values := range resp.Header {
		for _, value := range values {
			response += fmt.Sprintf("%s: %s\n", key, value)
		}
	}
	response += "\n"
	response += string(body)
	return response, nil
}
