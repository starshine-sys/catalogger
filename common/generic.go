package common

// There's no standard library package to deal with slices [grumble grumble]

// Contains returns whether `v` is in `slice`.
func Contains[T comparable](slice []T, v T) bool {
	for i := range slice {
		if slice[i] == v {
			return true
		}
	}
	return false
}
