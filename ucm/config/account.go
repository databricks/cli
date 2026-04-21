package config

// Account describes the Databricks account that hosts the metastore.
//
// M0 captures just the identifying fields; account-scoped operations
// (metastore CRUD, metastore assignment) land in M1+.
type Account struct {
	AccountId string `json:"account_id,omitempty"`
	Host      string `json:"host,omitempty"`
}
