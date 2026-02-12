package registry

// NewDefaultResolver creates the default ImageResolver for the platform.
// When built with containers_image_* build tags, this uses go.podman.io/image/v5.
// Without build tags, this returns nil (slow checks won't be available).
var NewDefaultResolver func() ImageResolver
