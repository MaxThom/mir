package mirv2

type serverRoutes struct {
	m *Mir
}

// Access all server routes
func (m *Mir) Server() *serverRoutes {
	return &serverRoutes{m: m}
}
