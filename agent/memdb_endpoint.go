package agent

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/consul/agent/consul"
)

// Memdb handles requests to get the memdb schema and db. This will open a port 8080
// and serve the data in UI
func (s *HTTPHandlers) Memdb(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	// only GET is allowed
	if req.Method != "GET" {
		return nil, MethodNotAllowedError{req.Method, []string{"GET"}}
	}
	// get query string param table
	table := req.URL.Query().Get("table")
	var server *consul.Server
	var ok bool
	if server, ok = s.agent.delegate.(*consul.Server); !ok {
		return nil, fmt.Errorf("agent is not a server")
	}
	store := server.FSM().State()
	schema := store.GetSchema()
	db := store.GetDB()
	keys := make([]string, 0, len(schema.Tables))
	// extract all the keys from the schema
	for k := range schema.Tables {
		keys = append(keys, k)
	}
	// sort keys alphabetically
	sort.Strings(keys)
	// check if table is in schema
	if _, ok := schema.Tables[table]; !ok {
		resp.Header().Set("Content-Type", "text/html; charset=utf-8")
		resp.Write([]byte("<ul>"))
		for _, k := range keys {
			resp.Write([]byte("<li>"))
			resp.Write([]byte("<a href=\"?table=" + k + "\">" + k + "</a>"))
			resp.Write([]byte("</li>"))
		}
		resp.Write([]byte("</ul>"))
		return nil, nil
	}
	tx := db.Txn(false)
	iter, err := tx.Get(table, "id")
	if err != nil {
		return nil, fmt.Errorf("error getting data for table %s: %v", table, err)
	}
	resp.Write([]byte("=== DATA FOR TABLE " + table + " ===\n"))
	for raw := iter.Next(); raw != nil; raw = iter.Next() {
		resp.Write([]byte("===\n"))
		b, err := json.MarshalIndent(raw, "", "  ")
		if err != nil {
			spew.Fdump(resp, raw)
			continue
		}
		resp.Write(b)
	}

	// extract dot if required
	dot := req.URL.Query().Get("dot")
	if dot != "" {
		resp.Write([]byte("=== 	RADIX TREE FOR " + table + " ===\n"))
		t := db.GetRoot()
		s := t.ToDot()
		resp.Write([]byte(s))
	}

	return nil, nil
}
