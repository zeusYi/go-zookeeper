// Copyright 2018-2019 The vogo Authors. All rights reserved.
// author: wongoo
// since: 2018/12/27
//

package zkclient

import (
	zk "github.com/zeusYi/go-zookeeper/go-lib-zk"
	"strings"
)

// StateAlive whether zk state alive
func StateAlive(state zk.State) bool {
	switch state {
	case zk.StateDisconnected, zk.StateAuthFailed, zk.StateConnecting:
		return false
	}

	return true
}

func IsZKInvalidErr(err error) bool {
	switch err {
	case zk.ErrInvalidACL, zk.ErrInvalidPath:
		return true
	default:
		return false
	}
}

func IsZKRecoverableErr(err error) bool {
	switch err {
	case zk.ErrClosing, zk.ErrConnectionClosed, zk.ErrSessionExpired, zk.ErrSessionMoved, zk.ErrNoServer:
		return true
	default:
		return false
	}
}

func ParentNode(path string) string {
	idx := strings.LastIndex(path, PathSplit)
	if idx > 0 {
		return path[:idx]
	}

	return ""
}

func PathJoin(nodes ...string) string {
	return strings.Join(nodes, PathSplit)
}
