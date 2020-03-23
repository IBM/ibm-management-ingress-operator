package handler

import (
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	defaultMemoryRequest resource.Quantity = resource.MustParse("150Mi")
	defaultCpuRequest    resource.Quantity = resource.MustParse("50m")

	defaultMemoryLimit resource.Quantity = resource.MustParse("256Mi")
	defaultCpuLimit    resource.Quantity = resource.MustParse("200m")
)
