package bot

// GetArgs returns a list of arguments from a payload, separated by space
// Quotes can be used to pass arguments with a space inside, such as `"hello world"`
func GetArgs(payload string) (args []string) {
	if payload == "" {
		return
	}

	args = make([]string, 0)
	j := 0
	quoted := false
	for i := 0; i < len(payload); i++ {
		switch payload[i] {
		// Space is the separator
		case ' ':
			// If we're in quoted mode, continue parsing
			if quoted {
				break
			}
			// Ignore sequential spaces
			if i <= j {
				j++
				break
			}
			val := payload[j:i]
			args = append(args, val)
			j = i + 1
		// Quotes
		case '"':
			if quoted {
				// End quote
				val := payload[j:i]
				args = append(args, val)
				j = i + 1
			} else {
				// Skip the open quote from the result
				j++
			}
			quoted = !quoted
		}
	}

	// Add the rest
	if j < len(payload) {
		args = append(args, payload[j:])
	}

	return
}
