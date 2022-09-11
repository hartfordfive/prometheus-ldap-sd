package store

type DataStore interface {
	Serialize(string) (string, error)
	IsReady() bool
	Shutdown()
}

var StoreInstance DataStore
