package netRpc

import (
	log "github.com/cihub/seelog"
)

type Common struct {
}

func (m *Common) SendMail(header string) bool {
	log.Debug(header)
	return false
}
