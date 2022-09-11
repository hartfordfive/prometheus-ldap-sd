package store

type DataStore interface {
	Serialize(string) (string, error)
	Shutdown()
}

var StoreInstance DataStore
