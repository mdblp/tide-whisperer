package detailederror

type DetailedError struct {
	Status          int    `json:"status"`  // Http status code
	ID              string `json:"id"`      // provided to user so that we can better track down issues
	Code            string `json:"code"`    // Code which may be used to translate the message to the final user
	Message         string `json:"message"` // Understandable message sent to the client
	InternalMessage string `json:"-"`       // used only for logging so we don't want to serialize it out
}

// set the internal message that we will use for logging
func (d DetailedError) SetInternalMessage(internal error) DetailedError {
	d.InternalMessage = internal.Error()
	return d
}
