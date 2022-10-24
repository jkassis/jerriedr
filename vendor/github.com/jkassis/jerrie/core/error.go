package core

// CompoundError is a series of errors
type CompoundError struct {
	error
	Errors []*error
}

// Add adds an error to the Compound Error
func (ce *CompoundError) Add(err *error) {
	if ce.Errors == nil {
		Log.Error("CompoundError.errors not initialized")
		return
	}
	ce.Errors = append(ce.Errors, err)
}

func (ce *CompoundError) Error() string {
	errString := ""
	for _, err := range ce.Errors {
		errString += (*err).Error()
	}
	return errString
}
