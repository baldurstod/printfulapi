package api

type MaxAgeError struct{}

func (e MaxAgeError) Error() string {
	return "Max age error"
}
