package handler

import (
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	defaultMemory     resource.Quantity = resource.MustParse("256Mi")
	defaultCpuRequest resource.Quantity = resource.MustParse("200m")
)
