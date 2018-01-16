package main

import "errors"

var (
	ErrInvalidHtml = errors.New("Error while parsing HTML")
	ErrBadResponse = errors.New("Wrong HTTP response code")
	ErrNoArgument  = errors.New("No argument")
	ErrInvalidURL  = errors.New("Invalid URL")
)
