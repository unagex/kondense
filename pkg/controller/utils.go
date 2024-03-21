package controller

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func containerToExclude(pod *corev1.Pod) []string {
	exclude := []string{}
	l, ok := pod.Annotations["unagex.com/kondense-exclude"]
	if ok {
		exclude = strings.Split(l, ",")
	}

	return exclude
}
