package mcp

import (
	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/momaek/henetdns/internal/app"
)

// NewServer creates and configures the MCP server with all DNS tools registered.
// rt must remain valid for the lifetime of the returned server.
func NewServer(rt *app.Runtime, username string) *server.MCPServer {
	s := server.NewMCPServer("henetdns", "0.1.0",
		server.WithToolCapabilities(false),
	)

	h := &handlers{rt: rt, username: username}

	s.AddTool(toolListZones(), h.listZones)
	s.AddTool(toolListRecords(), h.listRecords)
	s.AddTool(toolUpsertRecord(), h.upsertRecord)
	s.AddTool(toolDeleteRecord(), h.deleteRecord)

	return s
}

func toolListZones() mcpsdk.Tool {
	return mcpsdk.NewTool("list_zones",
		mcpsdk.WithDescription("List all DNS zones managed by this HE.net account. Uses local cache by default; set refresh=true to fetch live data from HE.net (requires active session)."),
		mcpsdk.WithBoolean("refresh",
			mcpsdk.Description("Bypass local cache and fetch from HE.net. Requires an active session."),
		),
	)
}

func toolListRecords() mcpsdk.Tool {
	return mcpsdk.NewTool("list_records",
		mcpsdk.WithDescription("List DNS records for a zone. Accepts zone name (e.g. example.com) or numeric zone ID. Uses local cache by default."),
		mcpsdk.WithString("zone",
			mcpsdk.Required(),
			mcpsdk.Description("Zone name (e.g. example.com) or numeric zone ID."),
		),
		mcpsdk.WithBoolean("refresh",
			mcpsdk.Description("Bypass local cache and fetch from HE.net. Requires an active session."),
		),
	)
}

func toolUpsertRecord() mcpsdk.Tool {
	return mcpsdk.NewTool("upsert_record",
		mcpsdk.WithDescription("Create a DNS record if an identical record does not already exist (idempotent). Requires an active session."),
		mcpsdk.WithString("zone",
			mcpsdk.Required(),
			mcpsdk.Description("Zone name or numeric zone ID."),
		),
		mcpsdk.WithString("type",
			mcpsdk.Required(),
			mcpsdk.Description("Record type: A, AAAA, TXT, CNAME, or MX."),
		),
		mcpsdk.WithString("name",
			mcpsdk.Required(),
			mcpsdk.Description("Record hostname or label (e.g. www or www.example.com)."),
		),
		mcpsdk.WithString("value",
			mcpsdk.Required(),
			mcpsdk.Description("Record value: IP address, CNAME target, TXT content, or MX hostname."),
		),
		mcpsdk.WithNumber("ttl",
			mcpsdk.Description("TTL in seconds. Defaults to 300 when omitted or 0."),
		),
		mcpsdk.WithNumber("priority",
			mcpsdk.Description("MX priority (integer >= 0). Only used when type is MX. Defaults to 10."),
		),
	)
}

func toolDeleteRecord() mcpsdk.Tool {
	return mcpsdk.NewTool("delete_record",
		mcpsdk.WithDescription("Delete an exact matching DNS record. All provided fields must match the existing record. Requires an active session."),
		mcpsdk.WithString("zone",
			mcpsdk.Required(),
			mcpsdk.Description("Zone name or numeric zone ID."),
		),
		mcpsdk.WithString("type",
			mcpsdk.Required(),
			mcpsdk.Description("Record type: A, AAAA, TXT, CNAME, or MX."),
		),
		mcpsdk.WithString("name",
			mcpsdk.Required(),
			mcpsdk.Description("Record hostname or label."),
		),
		mcpsdk.WithString("value",
			mcpsdk.Required(),
			mcpsdk.Description("Record value."),
		),
		mcpsdk.WithNumber("priority",
			mcpsdk.Description("MX priority. Required for exact-match deletion of MX records."),
		),
	)
}
