package engine

// Host is the game's door to the outside world (storage, network, events).
// Hosts: the WASM client bridges to localStorage + PocketBase; tests use MemHost.
type Host interface {
	Load(key string) []byte
	Save(key string, v []byte)
	// Emit publishes a named event with a JSON payload (e.g. "run", "label").
	Emit(event string, jsonPayload []byte)
}

type MemHost struct {
	Store  map[string][]byte
	Events []HostEvent
}
type HostEvent struct {
	Name    string
	Payload []byte
}

func NewMemHost() *MemHost                 { return &MemHost{Store: map[string][]byte{}} }
func (m *MemHost) Load(k string) []byte    { return m.Store[k] }
func (m *MemHost) Save(k string, v []byte) { m.Store[k] = append([]byte(nil), v...) }
func (m *MemHost) Emit(n string, p []byte) {
	m.Events = append(m.Events, HostEvent{n, append([]byte(nil), p...)})
}
