/*
Package fact exposes contextual information about the environment and the requests
being made to a server.  It works with the context package to store and retrieve information.

This package follows the general pattern in use for golang.org/x/net/context.  For each
fact named X, there are two methods provided: (1) a getter with the signature X(context) (X, bool),
and (2) a setter with the signature SetX(context, X) context.
*/
package fact
