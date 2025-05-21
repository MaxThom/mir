package mir_v1

import (
	"fmt"
	"time"
)

type Object struct {
	ApiVersion string `json:"apiVersion,omitempty" yaml:"apiVersion"`
	Kind       string `json:"kind,omitempty" yaml:"kind"`
	Meta       Meta   `json:"meta,omitempty" yaml:"meta"`
}

type ObjectUpdate struct {
	ApiVersion *string     `json:"apiVersion,omitempty" yaml:"apiVersion"`
	Kind       *string     `json:"kind,omitempty" yaml:"kind"`
	Meta       *MetaUpdate `json:"meta,omitempty" yaml:"meta"`
}

var (
	ObjectNameMissing error = fmt.Errorf("object name is missing")
)

func (o *Object) Validate() error {
	if o.Meta.Name == "" {
		return ObjectNameMissing
	}
	// if o.Meta == nil {
	// 	cdr.Meta = &common_apiv1.Meta{}
	// }
	if o.Meta.Namespace == "" {
		o.Meta.Namespace = "default"
	}
	return nil
}

type Meta struct {
	Name        string            `json:"name,omitempty" yaml:"name"`
	Namespace   string            `json:"namespace,omitempty" yaml:"namespace"`
	Labels      map[string]string `json:"labels,omitempty" yaml:"labels"`
	Annotations map[string]string `json:"annotations,omitempty" yaml:"annotations"`
}

type MetaUpdate struct {
	Name        *string            `json:"name,omitempty" yaml:"name"`
	Namespace   *string            `json:"namespace,omitempty" yaml:"namespace"`
	Labels      map[string]*string `json:"labels,omitempty" yaml:"labels"`
	Annotations map[string]*string `json:"annotations,omitempty" yaml:"annotations"`
}

type ObjectTarget struct {
	Names      []string
	Namespaces []string
	Labels     map[string]string
}

func (o ObjectTarget) HasNoTarget() bool {
	return len(o.Names) == 0 &&
		len(o.Namespaces) == 0 &&
		len(o.Labels) == 0
}

type DateFilter struct {
	To   time.Time
	From time.Time
}
