// Copyright 2016 ByteDance, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"github.com/bytedance/dbatman/config"
	"github.com/bytedance/dbatman/database/cluster"
	. "github.com/bytedance/dbatman/database/mysql"
	"net"
	"sync/atomic"
)

var DEFAULT_CAPABILITY uint32 = uint32(ClientLongPassword | ClientLongFlag |
	ClientConnectWithDB | ClientProtocol41 | ClientTransactions | ClientSecureConn)

var baseConnId uint32 = 10000

type Session struct {
	server *Server
	config *config.ProxyConfig
	user   *config.UserConfig

	connID    uint32
	status    uint32
	collation CollationId
	charset   string

	salt []byte

	cluster *cluster.Cluster
	fc      *MySQLServerConn

	closed bool
	db     string
}

func (s *Server) newSession(conn net.Conn) *Session {
	session := new(Session)

	session.server = s

	session.connID = atomic.AddUint32(&baseConnId, 1)

	session.status = uint32(StatusInAutocommit)

	session.salt, _ = RandomBuf(20)

	session.collation = DEFAULT_COLLATION_ID
	session.charset = DEFAULT_CHARSET

	session.fc = NewMySQLServerConn(session, conn)

	return session
}

func (session *Session) HandshakeWithFront() error {

	return session.fc.Handshake()

	// TODO set cluster with auth info
	// session.cluster = cluster.New(db)
}

func (session *Session) Run() error {

	for {
		data, err := session.front.ReadPacket()
		if err != nil {
			return err
		}

		if err := session.dispatch(data); err != nil {
			if err != mysql.ErrBadConn {
				session.writeError(err)
				return nil
			}

			log.Warnf("con[%d], dispatch error %s", c.connID, err.Error())
			return err
		}

		if session.closed {
			return
		}

		session.ResetSequence()
	}

	return nil
}
