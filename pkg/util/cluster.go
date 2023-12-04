package util

const (
	CustomEndpointProvidedAnnotation = "spectrocloud.com/custom-dns-provided"
)

func IsCustomEndpointPresent(annotations map[string]string) bool {
	_, ok := annotations[CustomEndpointProvidedAnnotation]
	return ok
}
