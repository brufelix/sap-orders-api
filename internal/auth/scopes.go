package auth

const (
	ScopeOrdersRead  = "orders.read"
	ScopeOrdersWrite = "orders.write"

	RoleOrdersReader = "Orders.Reader"
	RoleOrdersWriter = "Orders.Writer"
	RoleOrdersAdmin  = "Orders.Admin"
)

var roleScopes = map[string][]string{
	RoleOrdersReader: {ScopeOrdersRead},
	RoleOrdersWriter: {ScopeOrdersRead, ScopeOrdersWrite},
	RoleOrdersAdmin:  {ScopeOrdersRead, ScopeOrdersWrite},
}
