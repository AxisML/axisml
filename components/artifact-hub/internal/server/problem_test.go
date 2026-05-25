package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBindingError(t *testing.T) {
	v := validator.New()
	type body struct {
		Name string `validate:"required"`
	}
	veErr := v.Struct(body{})
	require.Error(t, veErr, "validator should fail on empty required field")

	var typeErr *json.UnmarshalTypeError
	require.Error(t, json.Unmarshal([]byte(`{"x":"not-an-int"}`), &struct {
		X int `json:"x"`
	}{}))
	_ = typeErr

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"validator.ValidationErrors", veErr, true},
		{
			"wrapped validator error",
			fmt.Errorf("bind: %w", veErr),
			true,
		},
		{
			"json.SyntaxError",
			func() error {
				return json.NewDecoder(strings.NewReader(`{"x"`)).Decode(&struct {
					X int `json:"x"`
				}{})
			}(),
			true,
		},
		{
			"json.UnmarshalTypeError",
			json.Unmarshal([]byte(`{"x":"abc"}`), &struct {
				X int `json:"x"`
			}{}),
			true,
		},
		{"io.EOF", io.EOF, true},
		{"io.ErrUnexpectedEOF", io.ErrUnexpectedEOF, true},
		{"plain error", errors.New("nope"), false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isBindingError(tc.err))
		})
	}
}
