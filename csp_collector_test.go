package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestHandlerForDisallowedMethods(t *testing.T) {
	disallowedMethods := []string{"GET", "DELETE", "PUT", "TRACE", "PATCH"}
	randomUrls := []string{"/", "/blah"}

	for _, method := range disallowedMethods {
		for _, url := range randomUrls {
			t.Run(method+url, func(t *testing.T) {
				request, err := http.NewRequest(method, url, nil)
				if err != nil {
					t.Fatalf("failed to create request: %v", err)
				}
				recorder := httptest.NewRecorder()
				handleViolationReport(recorder, request)

				response := recorder.Result()
				defer response.Body.Close()

				if response.StatusCode != http.StatusMethodNotAllowed {
					t.Errorf("expected HTTP status %v; got %v", http.StatusMethodNotAllowed, response.StatusCode)
				}
			})
		}
	}
}

func TestHandlerForAllowingHealthcheck(t *testing.T) {
	request, err := http.NewRequest("GET", "/_healthcheck", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	recorder := httptest.NewRecorder()

	handleViolationReport(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Errorf("expected HTTP status %v; got %v", http.StatusOK, response.StatusCode)
	}
}

func TestValidateViolationWithInvalidBlockedURIs(t *testing.T) {
	invalidBlockedURIs := []string{
		"resource://",
		"chromenull://",
		"chrome-extension://",
		"safari-extension://",
		"mxjscall://",
		"webviewprogressproxy://",
		"res://",
		"mx://",
		"safari-resource://",
		"chromeinvoke://",
		"chromeinvokeimmediate://",
		"mbinit://",
		"opera://",
		"localhost",
		"127.0.0.1",
		"none://",
		"about:blank",
		"android-webview",
		"ms-browser-extension",
		"wvjbscheme://__wvjb_queue_message__",
		"nativebaiduhd://adblock",
		"bdvideo://error",
	}

	for _, blockedURI := range invalidBlockedURIs {
		// Makes the test name more readable for the output.
		testName := strings.Replace(blockedURI, "://", "", -1)

		t.Run(testName, func(t *testing.T) {
			var rawReport = []byte(fmt.Sprintf(`{
				"csp-report": {
					"blocked-uri": "%s"
				}
			}`, blockedURI))

			var report CSPReport
			jsonErr := json.Unmarshal(rawReport, &report)
			if jsonErr != nil {
				fmt.Println("error:", jsonErr)
			}

			validateErr := validateViolation(report)
			if validateErr == nil {
				t.Errorf("expected error to be raised but it didn't")
			}

			if validateErr.Error() != fmt.Sprintf("blocked URI ('%s') is an invalid resource", blockedURI) {
				t.Errorf("expected error to include correct message string but it didn't")
			}
		})
	}
}

func TestValidateViolationWithValidBlockedURIs(t *testing.T) {
	var rawReport = []byte(`{
		"csp-report": {
			"blocked-uri": "https://google.com/example.css"
		}
	}`)

	var report CSPReport
	jsonErr := json.Unmarshal(rawReport, &report)
	if jsonErr != nil {
		fmt.Println("error:", jsonErr)
	}

	validateErr := validateViolation(report)
	if validateErr != nil {
		t.Errorf("expected error not be raised")
	}
}

func TestHandleViolationReportMultipleTypeStatusCode(t *testing.T) {
	// Discard the output we create from the calls here.
	log.SetOutput(ioutil.Discard)

	statusCodeValues := []interface{}{"200", 200}

	for _, statusCode := range statusCodeValues {
		t.Run(fmt.Sprintf("%T", statusCode), func(t *testing.T) {
			csp := CSPReport{
				CSPReportBody{
					StatusCode: statusCode,
				},
			}

			payload, err := json.Marshal(csp)
			if err != nil {
				t.Fatalf("failed to marshal JSON: %v", err)
			}

			request, err := http.NewRequest("POST", "/", bytes.NewBuffer(payload))
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			recorder := httptest.NewRecorder()
			handleViolationReport(recorder, request)

			response := recorder.Result()
			defer response.Body.Close()

			if response.StatusCode != http.StatusOK {
				t.Errorf("expected HTTP status %v; got %v", http.StatusOK, response.StatusCode)
			}
		})
	}
}
