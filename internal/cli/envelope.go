package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/geekjourneyx/md2x/internal/article"
)

const schemaVersion = "v1"

type Envelope struct {
	Success       bool        `json:"success"`
	SchemaVersion string      `json:"schema_version"`
	Status        string      `json:"status"`
	Code          string      `json:"code"`
	Message       string      `json:"message"`
	Data          interface{} `json:"data,omitempty"`
	Error         interface{} `json:"error,omitempty"`
}

type ExitError struct {
	Code        string
	Message     string
	Exit        int
	Err         error
	Diagnostics []article.Diagnostic
}

func (e *ExitError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Code
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*ExitError); ok {
		return exitErr.Exit
	}
	return 1
}

func failureEnvelope(err *ExitError) Envelope {
	code := err.Code
	if code == "" {
		code = "ERROR"
	}
	message := err.Error()
	errorData := map[string]interface{}{
		"code":    code,
		"message": message,
	}
	if len(err.Diagnostics) > 0 {
		errorData["diagnostics"] = err.Diagnostics
	}
	return Envelope{
		Success:       false,
		SchemaVersion: schemaVersion,
		Status:        "failed",
		Code:          code,
		Message:       message,
		Error:         errorData,
	}
}

func writeJSON(w io.Writer, envelope Envelope) error {
	return writeJSONValue(w, envelope)
}

func writeJSONValue(w io.Writer, value interface{}) error {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", out)
	return err
}
