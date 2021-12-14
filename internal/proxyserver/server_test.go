package proxyserver

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO(TESTS): add unit tests for other ProxyServer methods

// UNIT TESTS

// TestBodyValidation tests the validateRequestBody method on ProxyServer
func TestBodyValidation(t *testing.T) {

	type unitTestCase struct {
		body        string
		msg         string
		exact       bool
		insensitive bool
		allowed     bool
	}

	server := &ProxyServer{}

	for _, tCase := range []unitTestCase{
		// exact
		{body: `{"body": "bad_message"}`, msg: "bad_message", exact: true, insensitive: false, allowed: false},
		{body: `{"body": " bad_message"}`, msg: "bad_message", exact: true, insensitive: false, allowed: false},
		{body: `{"body": "bad_message "}`, msg: "bad_message", exact: true, insensitive: false, allowed: false},
		{body: `{"body": " bad_message "}`, msg: "bad_message", exact: true, insensitive: false, allowed: false},
		{body: `{"body": "bad_messages"}`, msg: "bad_message", exact: true, insensitive: false, allowed: true},
		{body: `{"body": "0bad_messages"}`, msg: "bad_message", exact: true, insensitive: false, allowed: true},
		// contains
		{body: `{"body": "bad_message"}`, msg: "bad_message", exact: false, insensitive: false, allowed: false},
		{body: `{"body": " bad_message"}`, msg: "bad_message", exact: false, insensitive: false, allowed: false},
		{body: `{"body": "bad_message "}`, msg: "bad_message", exact: false, insensitive: false, allowed: false},
		{body: `{"body": " bad_message "}`, msg: "bad_message", exact: false, insensitive: false, allowed: false},
		{body: `{"body": "bad_messages"}`, msg: "bad_message", exact: false, insensitive: false, allowed: false},
		{body: `{"body": "0bad_messages"}`, msg: "bad_message", exact: false, insensitive: false, allowed: false},
		// case-sensitive
		{body: `{"body": "BAD_MESSAGE"}`, msg: "bad_message", exact: true, insensitive: false, allowed: true},
		// case-insensitive
		{body: `{"body": "BAD_MESSAGE"}`, msg: "bad_message", exact: true, insensitive: true, allowed: false},
	} {
		t.Run(fmt.Sprintf("body=%s/message=%s/exact=%t/insensitive=%t/allowed=%t", tCase.body, tCase.msg, tCase.exact, tCase.insensitive, tCase.allowed),
			func(t *testing.T) {

				server.RejectWith = tCase.msg
				server.RejectExact = tCase.exact
				server.RejectInsensitive = tCase.insensitive

				err := server.validateRequestBody(tCase.body)

				if tCase.allowed {
					assert.NoError(t, err)
				} else {
					assert.Error(t, err)
					assert.Equal(t, err.Error(), "rejected because `"+tCase.msg+"` found within request body")
				}

			})
	}

}
