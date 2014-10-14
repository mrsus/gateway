package raft

import (
	"fmt"
	"log"

	"github.com/AnyPresence/gateway/db"
	"github.com/AnyPresence/gateway/model"
	goraft "github.com/goraft/raft"
)

func init() {
	log.Print("Registering Raft commands")
	goraft.RegisterCommand(&proxyEndpointDBCommand{})
	goraft.RegisterCommand(&libraryDBCommand{})
}

// DBWriteAction is a write action a data store can do.
type DBWriteAction string

const (
	// Insert actions insert items into the data store.
	Insert = "insert"
	// Update actions update items in the data store.
	Update = "update"
	// Delete actions delete items from the data store.
	Delete = "delete"
)

// DBCommand is a helper struct for Raft actions that modify the data store.
type DBCommand struct {
	Action DBWriteAction `json:"action"`
}

// Apply insterts the instance in the data store.
func (c *DBCommand) Apply(server goraft.Server, instance model.Model) (interface{}, error) {
	db := server.Context().(db.DB)
	switch c.Action {
	case Insert:
		return nil, db.Insert(instance)
	case Update:
		return nil, db.Update(instance)
	case Delete:
		return nil, db.Delete(instance, instance.ID())
	default:
		return nil, fmt.Errorf("Unknown action")
	}
}

// ProxyEndpointDBCommand is a DBCommand to modify ProxyEndpoint instances.
type proxyEndpointDBCommand struct {
	DBCommand `json:"command"`
	Instance  model.ProxyEndpoint `json:"instance"`
}

// ProxyEndpointDBCommand returns a new command to execute with the proxy endpoint.
func ProxyEndpointDBCommand(action DBWriteAction, instance model.ProxyEndpoint) *proxyEndpointDBCommand {
	return &proxyEndpointDBCommand{DBCommand: DBCommand{Action: action}, Instance: instance}
}

// CommandName is the name of the command in the Raft log.
func (c *proxyEndpointDBCommand) CommandName() string {
	return "ProxyEndpoint"
}

// Apply runs the DB action against the data store.
func (c *proxyEndpointDBCommand) Apply(server goraft.Server) (interface{}, error) {
	return c.DBCommand.Apply(server, c.Instance)
}

// LibraryDBCommand is a DBCommand to modify Library instances.
type libraryDBCommand struct {
	DBCommand `json:"command"`
	Instance  model.Library `json:"instance"`
}

// LibraryDBCommand returns a new command to execute with the proxy endpoint.
func LibraryDBCommand(action DBWriteAction, instance model.Library) *libraryDBCommand {
	return &libraryDBCommand{DBCommand: DBCommand{Action: action}, Instance: instance}
}

// CommandName is the name of the command in the Raft log.
func (c *libraryDBCommand) CommandName() string {
	return "Library"
}

// Apply runs the DB action against the data store.
func (c *libraryDBCommand) Apply(server goraft.Server) (interface{}, error) {
	return c.DBCommand.Apply(server, c.Instance)
}
