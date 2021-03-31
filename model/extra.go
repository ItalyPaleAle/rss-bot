package model

import "errors"

// The methods in this file are not auto-generated

// RouteCallback is the callback function for a given route
type RouteCallback func(m *InMessage) error

// MessageFromContent returns an OutMessage for the right kind of content
// This method supports a variety of struct pointers (such as *OutTextMessage, *OutPhotoMessage, *OutFileMessage, etc) as well as simple strings
func MessageFromContent(content interface{}) (*OutMessage, error) {
	// Resulting object
	out := &OutMessage{}

	// Ensure content is in the right struct depending on its type
	switch c := content.(type) {
	case *OutMessage:
		// Content is already a pointer to *OutMessage
		out = c
	case *OutMessage_Text:
	case *OutMessage_File:
	case *OutMessage_Photo:
		// Already in the right format
		out.Content = c
	case *OutTextMessage:
		// Text message
		out.Content = &OutMessage_Text{
			Text: c,
		}
	case *OutFileMessage:
		// File
		out.Content = &OutMessage_File{
			File: c,
		}
	case *OutPhotoMessage:
		// Photo
		out.Content = &OutMessage_Photo{
			Photo: c,
		}
	case string:
		// String
		out.Content = &OutMessage_Text{
			Text: &OutTextMessage{
				Text: c,
			},
		}
	default:
		return nil, errors.New("Invalid content argument")
	}

	return out, nil
}
