package sap

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/brufelix/sap-orders-api/internal/domain"
)

const XMLNamespace = "urn:sap:demand:update"

type DemandUpdate struct {
	XMLName     xml.Name   `xml:"DemandUpdate"`
	Xmlns       string     `xml:"xmlns,attr"`
	OrderNumber string     `xml:"OrderNumber"`
	Item        DemandItem `xml:"Item"`
}

type DemandItem struct {
	DemandCode   string `xml:"DemandCode"`
	Description  string `xml:"Description"`
	DeliveryDate string `xml:"DeliveryDate"`
	Status       string `xml:"Status"`
}

type SyncResult struct {
	XMLResponse string
	Success     bool
	Message     string
}

type Client interface {
	SyncDemandUpdate(ctx context.Context, rfcFunction, xmlPayload string) (*SyncResult, error)
}

func BuildDemandUpdateXML(orderNumber string, item domain.OrderItem) (string, error) {
	payload := DemandUpdate{
		Xmlns:       XMLNamespace,
		OrderNumber: orderNumber,
		Item: DemandItem{
			DemandCode:   item.DemandCode,
			Description:  item.Description,
			DeliveryDate: item.DeliveryDate.Format(time.DateOnly),
			Status:       string(item.Status),
		},
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	encoder := xml.NewEncoder(&buf)
	encoder.Indent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		return "", fmt.Errorf("encode demand update xml: %w", err)
	}

	return buf.String(), nil
}
