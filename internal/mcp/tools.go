package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/momaek/henetdns/internal/app"
	"github.com/momaek/henetdns/internal/errs"
	"github.com/momaek/henetdns/internal/henet"
)

type handlers struct {
	rt       *app.Runtime
	username string
}

// toolErr converts any application error to an MCP tool error result.
// Auth errors get a human-actionable message. All errors stay as tool-level
// errors (IsError=true) rather than protocol-level errors, keeping the server alive.
func toolErr(err error) *mcpsdk.CallToolResult {
	if errors.Is(err, errs.ErrAuthRequired) {
		return mcpsdk.NewToolResultError(
			"No active session. Run 'henetdns login' to authenticate, then retry the tool call.")
	}
	return mcpsdk.NewToolResultError(err.Error())
}

func jsonText(v any) (*mcpsdk.CallToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal result: %w", err)
	}
	return mcpsdk.NewToolResultText(string(b)), nil
}

func (h *handlers) listZones(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	refresh := req.GetBool("refresh", false)

	if refresh {
		if err := h.rt.Auth.EnsureSession(ctx, h.username); err != nil {
			return toolErr(err), nil
		}
		zones, err := h.rt.HENet.ListZones(ctx)
		if err != nil {
			return toolErr(err), nil
		}
		return jsonText(zones)
	}

	zones, err := h.rt.HENet.ListZonesFromCache(ctx)
	if err != nil {
		return toolErr(err), nil
	}
	if len(zones) > 0 {
		return jsonText(zones)
	}
	if err := h.rt.Auth.EnsureSession(ctx, h.username); err != nil {
		return toolErr(err), nil
	}
	zones, err = h.rt.HENet.ListZones(ctx)
	if err != nil {
		return toolErr(err), nil
	}
	return jsonText(zones)
}

func (h *handlers) listRecords(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	zone, err := req.RequireString("zone")
	if err != nil {
		return toolErr(fmt.Errorf("zone is required: %w", errs.ErrInvalidInput)), nil
	}
	refresh := req.GetBool("refresh", false)

	ensured := false
	ensureSession := func() error {
		if ensured {
			return nil
		}
		if err := h.rt.Auth.EnsureSession(ctx, h.username); err != nil {
			return err
		}
		ensured = true
		return nil
	}

	if refresh {
		if err := ensureSession(); err != nil {
			return toolErr(err), nil
		}
		zoneID, err := h.rt.HENet.ResolveZoneID(ctx, zone)
		if err != nil {
			return toolErr(err), nil
		}
		records, err := h.rt.HENet.ListRecords(ctx, zoneID)
		if err != nil {
			return toolErr(err), nil
		}
		return jsonText(records)
	}

	zoneID, err := h.rt.HENet.ResolveZoneIDFromCache(ctx, zone)
	if err != nil {
		if err2 := ensureSession(); err2 != nil {
			return toolErr(err2), nil
		}
		zoneID, err = h.rt.HENet.ResolveZoneID(ctx, zone)
		if err != nil {
			return toolErr(err), nil
		}
	}

	records, err := h.rt.HENet.ListRecordsFromCache(ctx, zoneID)
	if err != nil {
		return toolErr(err), nil
	}
	if len(records) > 0 {
		return jsonText(records)
	}
	if err := ensureSession(); err != nil {
		return toolErr(err), nil
	}
	records, err = h.rt.HENet.ListRecords(ctx, zoneID)
	if err != nil {
		return toolErr(err), nil
	}
	return jsonText(records)
}

func (h *handlers) upsertRecord(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	zone, err := req.RequireString("zone")
	if err != nil {
		return toolErr(fmt.Errorf("zone is required: %w", errs.ErrInvalidInput)), nil
	}
	rrType, err := req.RequireString("type")
	if err != nil {
		return toolErr(fmt.Errorf("type is required: %w", errs.ErrInvalidInput)), nil
	}
	name, err := req.RequireString("name")
	if err != nil {
		return toolErr(fmt.Errorf("name is required: %w", errs.ErrInvalidInput)), nil
	}
	value, err := req.RequireString("value")
	if err != nil {
		return toolErr(fmt.Errorf("value is required: %w", errs.ErrInvalidInput)), nil
	}

	ttl := req.GetInt("ttl", 0)
	priority := req.GetInt("priority", 0)
	hasPriority := priority > 0

	input := henet.RecordInput{
		Type:        rrType,
		Name:        name,
		Value:       value,
		TTL:         ttl,
		Priority:    priority,
		HasPriority: hasPriority,
	}

	if err := h.rt.Auth.EnsureSession(ctx, h.username); err != nil {
		return toolErr(err), nil
	}
	zoneID, err := h.rt.HENet.ResolveZoneIDCachedFirst(ctx, zone)
	if err != nil {
		return toolErr(err), nil
	}
	if err := h.rt.HENet.UpsertRecord(ctx, zoneID, input); err != nil {
		return toolErr(err), nil
	}
	return mcpsdk.NewToolResultText(`{"message":"upsert ok"}`), nil
}

func (h *handlers) deleteRecord(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	zone, err := req.RequireString("zone")
	if err != nil {
		return toolErr(fmt.Errorf("zone is required: %w", errs.ErrInvalidInput)), nil
	}
	rrType, err := req.RequireString("type")
	if err != nil {
		return toolErr(fmt.Errorf("type is required: %w", errs.ErrInvalidInput)), nil
	}
	name, err := req.RequireString("name")
	if err != nil {
		return toolErr(fmt.Errorf("name is required: %w", errs.ErrInvalidInput)), nil
	}
	value, err := req.RequireString("value")
	if err != nil {
		return toolErr(fmt.Errorf("value is required: %w", errs.ErrInvalidInput)), nil
	}

	priority := req.GetInt("priority", 0)
	hasPriority := priority > 0

	input := henet.RecordInput{
		Type:        rrType,
		Name:        name,
		Value:       value,
		Priority:    priority,
		HasPriority: hasPriority,
	}

	if err := h.rt.Auth.EnsureSession(ctx, h.username); err != nil {
		return toolErr(err), nil
	}
	zoneID, err := h.rt.HENet.ResolveZoneIDCachedFirst(ctx, zone)
	if err != nil {
		return toolErr(err), nil
	}
	if err := h.rt.HENet.DeleteRecord(ctx, zoneID, input); err != nil {
		return toolErr(err), nil
	}
	return mcpsdk.NewToolResultText(`{"message":"delete ok"}`), nil
}
