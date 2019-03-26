package io3mf

import (
	"encoding/xml"
	"errors"
	"strconv"

	"github.com/gofrs/uuid"
	go3mf "github.com/qmuntal/go3mf"
)

type objectDecoder struct {
	r                               *Reader
	obj                             go3mf.ObjectResource
	colorMapping                    *colorMapping
	texCoordMapping                 *texCoordMapping
	defaultPropID, defaultPropIndex uint64
}

func (d *objectDecoder) Decode(x xml.TokenReader, se xml.StartElement) error {
	if err := d.parseAttr(se.Attr); err != nil {
		return err
	}
	return d.parseContent(x)
}

func (d *objectDecoder) parseContent(x xml.TokenReader) error {
	for {
		t, err := x.Token()
		if err != nil {
			return err
		}
		switch tp := t.(type) {
		case xml.StartElement:
			var err error
			if tp.Name.Space == nsCoreSpec {
				if tp.Name.Local == attrMesh {
					err = d.parseMesh(x, tp)
				} else if tp.Name.Local == attrComponents {
					err = d.parseComponents(x, tp)
				}
			}
			if err != nil {
				return err
			}
		case xml.EndElement:
			if tp.Name.Space == nsCoreSpec && tp.Name.Local == attrObject {
				return nil
			}
		}
	}
}

func (d *objectDecoder) parseMesh(x xml.TokenReader, se xml.StartElement) error {
	d.r.progress.pushLevel(1, 0)
	md := meshDecoder{
		r: d.r, resource: go3mf.MeshResource{ObjectResource: d.obj},
		colorMapping: d.colorMapping, texCoordMapping: d.texCoordMapping,
		defaultPropID: d.defaultPropID, defaultPropIndex: d.defaultPropIndex,
	}
	if err := md.Decode(x, se); err != nil {
		return err
	}
	d.r.progress.popLevel()
	return nil
}

func (d *objectDecoder) parseComponents(x xml.TokenReader, se xml.StartElement) error {
	if d.defaultPropID != 0 {
		d.r.Warnings = append(d.r.Warnings, &ReadError{InvalidOptionalValue, "go3mf: a components object must not have a default PID"})
	}
	cd := componentsDecoder{r: d.r, components: go3mf.ComponentsResource{ObjectResource: d.obj}}
	return cd.Decode(x, se)
}

func (d *objectDecoder) parseAttr(attrs []xml.Attr) (err error) {
	for _, a := range attrs {
		switch a.Name.Space {
		case nsProductionSpec:
			if a.Name.Local == attrProdUUID {
				if d.obj.UUID != "" {
					d.r.Warnings = append(d.r.Warnings, &ReadError{InvalidMandatoryValue, "go3mf: duplicated object resource uuid attribute"})
				}
				if _, err = uuid.FromString(a.Value); err != nil {
					err = errors.New("go3mf: object resource uuid is not valid")
				} else {
					d.obj.UUID = a.Value
				}
			}
		case nsSliceSpec:
			err = d.parseSliceAttr(a)
		case "":
			err = d.parseCoreAttr(a)
		}
		if err != nil {
			break
		}
	}
	return
}
func (d *objectDecoder) parseCoreAttr(a xml.Attr) (err error) {
	switch a.Name.Local {
	case attrID:
		if d.obj.ID != 0 {
			err = errors.New("go3mf: duplicated object resource id attribute")
		} else {
			d.obj.ID, err = strconv.ParseUint(a.Value, 10, 64)
			if err != nil {
				err = errors.New("go3mf: object resource id is not valid")
			}
		}
	case attrType:
		var ok bool
		d.obj.ObjectType, ok = go3mf.NewObjectType(a.Value)
		if !ok {
			d.r.Warnings = append(d.r.Warnings, &ReadError{InvalidOptionalValue, "go3mf: object resource type is not valid"})
		}
	case attrThumbnail:
		d.obj.Thumbnail = a.Value
	case attrName:
		d.obj.Name = a.Value
	case attrPartNumber:
		d.obj.PartNumber = a.Value
	case attrPID:
		d.defaultPropID, err = strconv.ParseUint(a.Value, 10, 64)
		if err != nil {
			err = errors.New("go3mf: object resource pid is not valid")
		}
	case attrPIndex:
		d.defaultPropIndex, err = strconv.ParseUint(a.Value, 10, 64)
		if err != nil {
			err = errors.New("go3mf: object resource ºpindex is not valid")
		}
	}
	return
}

func (d *objectDecoder) parseSliceAttr(a xml.Attr) (err error) {
	switch a.Name.Local {
	case attrSliceRefID:
		if d.obj.SliceStackID != 0 {
			d.r.Warnings = append(d.r.Warnings, &ReadError{InvalidOptionalValue, "go3mf: duplicated object resource slicestackid attribute"})
		}
		d.obj.SliceStackID, err = strconv.ParseUint(a.Value, 10, 64)
		if err != nil {
			err = errors.New("go3mf: object resource slicestackid is not valid")
		}
	case attrMeshRes:
		var ok bool
		d.obj.SliceResoultion, ok = go3mf.NewSliceResolution(a.Value)
		if !ok {
			err = errors.New("go3mf: object resource sliceresolution is not valid")
		}
	}
	return
}

type componentsDecoder struct {
	r          *Reader
	components go3mf.ComponentsResource
}

func (d *componentsDecoder) Decode(x xml.TokenReader, se xml.StartElement) error {
	for {
		t, err := x.Token()
		if err != nil {
			return err
		}
		switch tp := t.(type) {
		case xml.StartElement:
			if tp.Name.Space == nsCoreSpec && tp.Name.Local == attrComponent {
				if err := d.parseComponent(tp.Attr); err != nil {
					return err
				}
			}
		case xml.EndElement:
			if tp.Name.Space == nsCoreSpec && tp.Name.Local == attrComponents {
				d.r.addResource(&d.components)
				return nil
			}
		}
	}
}

func (d *componentsDecoder) parseComponent(attrs []xml.Attr) (err error) {
	var component go3mf.Component
	var path string
	var objectID uint64
	for _, a := range attrs {
		switch a.Name.Space {
		case nsProductionSpec:
			if a.Name.Local == attrProdUUID {
				if component.UUID != "" {
					d.r.Warnings = append(d.r.Warnings, &ReadError{InvalidMandatoryValue, "go3mf: duplicated component uuid attribute"})
				}
				if _, err = uuid.FromString(a.Value); err != nil {
					err = errors.New("go3mf: component uuid is not valid")
				} else {
					component.UUID = a.Value
				}
			} else if a.Name.Local == attrPath {
				if path != "" {
					d.r.Warnings = append(d.r.Warnings, &ReadError{InvalidMandatoryValue, "go3mf: duplicated component path attribute"})
				}
				path = a.Value
			}
		case "":
			if a.Name.Local == attrObjectID {
				if objectID != 0 {
					err = errors.New("go3mf: duplicated component objectid attribute")
				}
				objectID, err = strconv.ParseUint(a.Value, 10, 64)
				if err != nil {
					err = errors.New("go3mf: component id is not valid")
				}
			} else if a.Name.Local == attrTransform {
				component.Transform, err = strToMatrix(a.Value)
			}
		}
		if err != nil {
			break
		}
	}
	if component.UUID == "" && d.r.namespaceRegistered(nsProductionSpec) {
		d.r.Warnings = append(d.r.Warnings, &ReadError{MissingMandatoryValue, "go3mf: a UUID for a component is missing"})
	}
	if path == "" {
		path = d.r.Model.Path
	}
	resource, ok := d.r.Model.FindResource(objectID, path)
	if !ok {
		err = errors.New("go3mf: could not find component object")
	}
	component.Object, ok = resource.(go3mf.Object)
	if !ok {
		return errors.New("go3mf: could not find component object")
	}
	d.components.Components = append(d.components.Components, &component)
	return
}
