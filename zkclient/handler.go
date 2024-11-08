// Copyright 2018-2019 The vogo Authors. All rights reserved.
// author: wongoo
// since: 2018/12/27
//

package zkclient

import (
	"errors"
	"github.com/vogo/logger"
	"github.com/zeusYi/go-zookeeper/go-lib-zk"
	"io"
	"reflect"
)

type valueHandler struct {
	path        string
	value       reflect.Value
	codec       Codec
	listenAsync bool
	listener    ValueListener
}

func (cli *Client) newValueHandler(path string, obj interface{}, codec Codec,
	watchOnly bool,
	listener ValueListener) (*valueHandler, error) {
	if path == "" {
		return nil, errors.New("path required")
	}

	typ := reflect.TypeOf(obj)
	if typ.Kind() != reflect.Ptr {
		return nil, errors.New("pointer object required")
	}

	if typ.Elem().Kind() == reflect.Ptr {
		return nil, errors.New("not support multiple level pointer object")
	}

	if codec == nil {
		return nil, errors.New("codec required")
	}

	if watchOnly && listener == nil {
		return nil, errors.New("listener required when watch only")
	}

	// set json value type
	if jsonCodec, ok := codec.(*JSONCodec); ok {
		jsonCodec.typ = typ.Elem()
	}

	handler := &valueHandler{
		path:        path,
		codec:       codec,
		listenAsync: cli.listenAsync,
		listener:    listener,
	}

	if !watchOnly {
		handler.value = reflect.ValueOf(obj)
	}

	return handler, nil
}

func (h *valueHandler) Encode() ([]byte, error) {
	if h.value == nilValue {
		return nil, nil
	}

	return h.codec.Encode(h.value.Interface())
}

func (h *valueHandler) Decode(stat *zk.Stat, data []byte) error {
	v, err := h.codec.Decode(data)
	if err != nil {
		return err
	}

	if h.value != nilValue {
		h.value.Elem().Set(reflect.ValueOf(v).Elem())
	}

	if h.listener != nil {
		f := func() {
			h.listener.Update(h.path, stat, h.value.Interface())
		}

		if h.listenAsync {
			go f()
		} else {
			f()
		}
	}

	return nil
}

// SetTo set value in zookeeper
func (h *valueHandler) SetTo(cli *Client, path string) error {
	bytes, err := h.Encode()
	if err != nil {
		return err
	}

	return cli.SetRawValue(path, bytes)
}

func (h *valueHandler) Path() string {
	return h.path
}

func (h *valueHandler) Handle(w *Watcher, evt *zk.Event) (<-chan zk.Event, error) {
	if evt != nil && evt.Type == zk.EventNodeDeleted {
		logger.Infof("zk watcher [%s] node deleted", h.path)

		if h.listener != nil {
			h.listener.Delete(h.path)
		}

		return nil, nil
	}

	data, stat, wch, err := w.client.Conn().GetW(h.path)
	if err != nil {
		if err == zk.ErrNoNode {
			data, err = h.Encode()
			if err != nil {
				return nil, err
			}

			if setErr := w.client.SetRawValue(h.path, data); setErr != nil {
				return nil, setErr
			}

			data, _, wch, err = w.client.Conn().GetW(h.path)
		}

		if err != nil {
			return nil, err
		}
	}

	if data == nil {
		// ignore nil config
		return wch, nil
	}

	if err := h.Decode(stat, data); err != nil {
		if err == io.EOF {
			return wch, nil // ignore nil data
		}

		logger.Warnf("zk failed to parse %s: %v", h.path, err)

		return wch, nil
	}

	return wch, nil
}
