package hold

import (
	"bytes"
	"encoding/gob"
)

// EncodeFunc is a function for encoding a value into bytes
type EncodeFunc func(value interface{}) ([]byte, error)

// DecodeFunc is a function for decoding a value from bytes
type DecodeFunc func(data []byte, value interface{}) error

// DefaultEncode is the default encoding func for hold (Gob)
func DefaultEncode(value interface{}) ([]byte, error) {
	var buff bytes.Buffer

	en := gob.NewEncoder(&buff)

	err := en.Encode(value)
	if err != nil {
		return nil, err
	}

	return buff.Bytes(), nil
}

// DefaultDecode is the default decoding func for hold (Gob)
func DefaultDecode(data []byte, value interface{}) error {
	var buff bytes.Buffer
	de := gob.NewDecoder(&buff)

	_, err := buff.Write(data)
	if err != nil {
		return err
	}

	return de.Decode(value)
}

// encodeKey encodes key values with a type prefix which allows multiple different types
// to exist in the badger DB
func (s *Store) encodeKey(key interface{}, typeName string) ([]byte, error) {
	encoded, err := s.encode(key)
	if err != nil {
		return nil, err
	}

	return append(typePrefix(typeName), encoded...), nil
}

// decodeKey decodes the key value and removes the type prefix
func (s *Store) decodeKey(data []byte, key interface{}, typeName string) error {
	return s.decode(data[len(typePrefix(typeName)):], key)
}
