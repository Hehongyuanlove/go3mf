package go3mf

import (
	"encoding/xml"
	"errors"
	"fmt"
	"image/color"
	"strings"

	specerr "github.com/qmuntal/go3mf/errors"
)

// NodeDecoder defines the minimum contract to decode a 3MF node.
type NodeDecoder interface {
	Start([]xml.Attr)
	Text([]byte)
	Child(xml.Name) NodeDecoder
	End()
	SetScanner(*Scanner)
}

type baseDecoder struct {
	Scanner *Scanner
}

func (d *baseDecoder) Start([]xml.Attr)           {}
func (d *baseDecoder) Text([]byte)                {}
func (d *baseDecoder) Child(xml.Name) NodeDecoder { return nil }
func (d *baseDecoder) End()                       {}
func (d *baseDecoder) SetScanner(s *Scanner)      { d.Scanner = s }

// A Scanner is a 3mf model file scanning state machine.
type Scanner struct {
	Resources        Resources
	BuildItems       []*Item
	ModelPath        string
	IsRoot           bool
	ResourceID       uint32
	Err              specerr.List
	extensionDecoder map[string]SpecDecoder
	contex           []xml.Name
}

func (s *Scanner) namespace(local string) (string, bool) {
	for _, ext := range s.extensionDecoder {
		if ext.Local() == local {
			return ext.Namespace(), true
		}
	}
	return "", false
}

// AddAsset adds a new resource to the resource cache.
func (s *Scanner) AddAsset(r Asset) {
	s.Resources.Assets = append(s.Resources.Assets, r)
	s.ResourceID = 0
}

// AddObject adds a new resource to the resource cache.
func (s *Scanner) AddObject(r *Object) {
	s.Resources.Objects = append(s.Resources.Objects, r)
	s.ResourceID = 0
}

// InvalidAttr adds the error to the errors.
// Returns false if scanning cannot continue.
func (s *Scanner) InvalidAttr(attr string, required bool) {
	ct := make([]string, len(s.contex))
	ct[0] = s.ModelPath
	for i, s := range s.contex[1:] {
		ct[i+1] = s.Local
	}
	if s.IsRoot {
		ct = ct[1:] // don't add path in case happend in root file
	}
	specerr.Append(&s.Err, &specerr.ParseFieldError{
		Context: strings.Join(ct, "@"), Name: attr, ResourceID: s.ResourceID, Required: required,
	})
}

// ParseRGBA parses s as a RGBA color.
func ParseRGBA(s string) (c color.RGBA, err error) {
	var errInvalidFormat = errors.New("gltf: invalid color format")

	if len(s) == 0 || s[0] != '#' {
		return c, errInvalidFormat
	}

	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		err = errInvalidFormat
		return 0
	}

	switch len(s) {
	case 9:
		c.R = hexToByte(s[1])<<4 + hexToByte(s[2])
		c.G = hexToByte(s[3])<<4 + hexToByte(s[4])
		c.B = hexToByte(s[5])<<4 + hexToByte(s[6])
		c.A = hexToByte(s[7])<<4 + hexToByte(s[8])
	case 7:
		c.R = hexToByte(s[1])<<4 + hexToByte(s[2])
		c.G = hexToByte(s[3])<<4 + hexToByte(s[4])
		c.B = hexToByte(s[5])<<4 + hexToByte(s[6])
		c.A = 0xff
	default:
		err = errInvalidFormat
	}
	return
}

// FormatRGBA returns the color as a hex string with the format #rrggbbaa.
func FormatRGBA(c color.RGBA) string {
	if c.A == 255 {
		return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", c.R, c.G, c.B, c.A)
}
