package utils

import (
	"github.com/imdario/mergo"
	v1 "k8s.io/api/core/v1"
)

func MergeContainerPorts(dst []v1.ContainerPort, src []v1.ContainerPort) []v1.ContainerPort {
	resultMap := map[string]v1.ContainerPort{}
	for _, dstItem := range dst {
		resultMap[dstItem.Name] = dstItem
	}
	for _, srcItem := range src {
		if val, ok := resultMap[srcItem.Name]; ok {
			_ = mergo.Merge(&val, srcItem, mergo.WithOverride)
			resultMap[srcItem.Name] = val
		} else {
			resultMap[srcItem.Name] = srcItem
		}
	}
	var results []v1.ContainerPort
	for _, item := range resultMap {
		results = append(results, item)
	}
	return results
}

func MergeVolumeMounts(dst []v1.VolumeMount, src []v1.VolumeMount) []v1.VolumeMount {
	resultMap := map[string]v1.VolumeMount{}
	for _, dstItem := range dst {
		resultMap[dstItem.Name] = dstItem
	}
	for _, srcItem := range src {
		if val, ok := resultMap[srcItem.Name]; ok {
			_ = mergo.Merge(&val, srcItem, mergo.WithOverride)
			resultMap[srcItem.Name] = val
		} else {
			resultMap[srcItem.Name] = srcItem
		}
	}
	var results []v1.VolumeMount
	for _, item := range resultMap {
		results = append(results, item)
	}
	return results
}
