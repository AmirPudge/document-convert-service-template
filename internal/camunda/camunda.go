package camunda

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type CamundaClient struct {
	baseURL     string
	messageName string
	http        *http.Client
}

func NewCamundaClient(baseURL, messageName string) *CamundaClient {
	return &CamundaClient{
		baseURL:     strings.TrimSpace(baseURL),
		messageName: messageName,
		http:        &http.Client{Timeout: 60 * time.Second},
	}
}

type CamundaMessage struct {
	Value any    `json:"value"`
	Type  string `json:"type"`
}

type CamundaPayload struct {
	MessageName      string                    `json:"messageName"`
	CorrelationKeys  map[string]CamundaMessage `json:"correlationKeys"`
	ProcessVariables map[string]CamundaMessage `json:"processVariables,omitempty"`
}

func (c *CamundaClient) SendMessage(ctx context.Context, correlationKey, requestID, pdfS3Key string) error {
	payload := CamundaPayload{
		MessageName:      c.messageName,
		CorrelationKeys:  map[string]CamundaMessage{correlationKey: {Value: requestID, Type: "string"}},
		ProcessVariables: map[string]CamundaMessage{"requestID": {Value: requestID, Type: "string"}, "pdfS3Key": {Value: pdfS3Key, Type: "string"}},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal camunda payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/engine-rest/message", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build camunda request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("camunda request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("camunda returned %d: %s", resp.StatusCode, respBody)
	}

	return nil
}
