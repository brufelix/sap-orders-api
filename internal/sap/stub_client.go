package sap

import (
	"context"
	"fmt"
	"log/slog"
)

// StubClient simulates SAP RFC calls during local development.
// Replace with an RFC-backed implementation when SAP NW RFC SDK is available.
type StubClient struct {
	logger *slog.Logger
}

func NewStubClient(logger *slog.Logger) *StubClient {
	return &StubClient{logger: logger}
}

func (c *StubClient) SyncDemandUpdate(ctx context.Context, rfcFunction, xmlPayload string) (*SyncResult, error) {
	_ = ctx

	c.logger.Info("sap stub: calling RFC",
		"rfc_function", rfcFunction,
		"payload_size", len(xmlPayload),
	)

	response := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<SapResponse xmlns="urn:sap:demand:response">
  <RFCFunction>%s</RFCFunction>
  <Status>OK</Status>
  <Message>Demand update accepted</Message>
</SapResponse>`, rfcFunction)

	return &SyncResult{
		XMLResponse: response,
		Success:     true,
		Message:     "Demand update accepted",
	}, nil
}
